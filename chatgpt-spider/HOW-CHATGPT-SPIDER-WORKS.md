# How chatgpt-spider Works 🕷️

A deep-dive technical guide into the architecture, design decisions, and mechanics of **`chatgpt-spider`**—a native Go automation engine and OpenAI-compatible REST server for ChatGPT.

---

## Table of Contents
- [1. Motivation & Philosophy](#1-motivation--philosophy)
- [2. System Architecture](#2-system-architecture)
- [3. Chrome DevTools Protocol (CDP) & Stealth Automation](#3-chrome-devtools-protocol-cdp--stealth-automation)
- [4. The Core Spider Engine (`internal/spider`)](#4-the-core-spider-engine-internalspider)
- [5. Real-Time Token Streaming Mechanics (`internal/conversation`)](#5-real-time-token-streaming-mechanics-internalconversation)
- [6. Concurrency Architecture & Deadlock Prevention](#6-concurrency-architecture--deadlock-prevention)
- [7. Process Lifecycle & Windows Lockfile Resilience](#7-process-lifecycle--windows-lockfile-resilience)
- [8. OpenAI REST API Server (`internal/server`)](#8-openai-rest-api-server-internalserver)
- [9. Session Persistence & Credential Security (`internal/auth`)](#9-session-persistence--credential-security-internalauth)
- [10. Embedding as a Go Library](#10-embedding-as-a-go-library)

---

## 1. Motivation & Philosophy

Legacy browser automation scripts (such as Python Selenium scripts) suffer from critical limitations:
1. **Webdriver Bottlenecks**: Requiring matching `chromedriver.exe` binaries that break whenever Chrome auto-updates.
2. **Cloudflare & Bot Detection**: Standard Selenium webdrivers expose detectable properties (`navigator.webdriver = true`, missing plugins, unmasked automation flags) that trigger Cloudflare Turnstile captchas and bot challenges.
3. **Fragile DOM Polling**: Polling transient, obfuscated CSS classnames (like `button.text-token-text-tertiary`) that break across ChatGPT UI deploys.
4. **Heavy Resource Footprint**: Large Python runtime overhead with no clean way to expose standardized LLM API endpoints.

**`chatgpt-spider`** was engineered with four guiding principles:
* **Zero External Drivers**: Communicate directly with Chromium over native WebSocket via the Chrome DevTools Protocol (CDP).
* **DOM-Native Real-Time Streaming**: Extract tokens chunk-by-chunk directly from the live DOM markdown containers as they render, completely bypassing Cloudflare network blocks.
* **Modular Core Engine**: A decoupled Go package (`internal/spider`) that can be used from the CLI, embedded into other Go microservices, or run as a local REST API server.
* **OpenAI Drop-In Compatibility**: Expose standard `/v1/chat/completions` supporting Server-Sent Events (SSE) streaming so any OpenAI-compatible tool (Cursor, LangChain, Python SDK) works out of the box.

---

## 2. System Architecture

```
                                  ┌────────────────────────────────────────────────────────┐
                                  │                     Client Layer                       │
                                  │  • Terminal CLI (prompt, chat, batch)                  │
                                  │  • Cursor / LangChain / OpenAI SDK                     │
                                  │  • External HTTP Webhooks                              │
                                  └───────────────────────────┬────────────────────────────┘
                                                              │
                                                              ▼
                                  ┌────────────────────────────────────────────────────────┐
                                  │                   Presentation Layer                   │
                                  │  ┌───────────────────────┐  ┌───────────────────────┐  │
                                  │  │  CLI Commands (cmd/)  │  │  REST Server (server) │  │
                                  │  │  prompt, chat, batch  │  │  /v1/chat/completions │  │
                                  │  └───────────┬───────────┘  └───────────┬───────────┘  │
                                  └──────────────┼──────────────────────────┼──────────────┘
                                                 │                          │
                                                 └────────────┬─────────────┘
                                                              │
                                                              ▼
                                  ┌────────────────────────────────────────────────────────┐
                                  │               Modular Core Engine (spider)             │
                                  │  • Engine struct (Browser instance, mutex locks)       │
                                  │  • Concurrency Controller (mu vs convMu)               │
                                  │  • Thread Session State (currentConv ID)               │
                                  └───────────────────────────┬────────────────────────────┘
                                                              │
                                  ┌───────────────────────────┴────────────────────────────┐
                                  ▼                                                        ▼
┌────────────────────────────────────────────────────────┐       ┌────────────────────────────────────────────────────────┐
│             Conversation Automation Layer              │       │                 Browser Lifecycle Layer                │
│                 (internal/conversation)                │       │                   (internal/browser)                   │
│  • Prompt Injection (#prompt-textarea)                 │       │  • go-rod Launcher & Stealth Injection                │
│  • CountAssistantMessages (baseline turn tracker)      │       │  • Auto-Sanitize Windows Lockfiles                     │
│  • StreamCompletion (polling delta streamer)           │       │  • Profile Persistence (~/.chatgpt-spider/profile)     │
│  • GetConversationHistory (DOM extractor)              │       │  • Graceful Signal Trapping (SIGINT / launcher.Kill)   │
└─────────────────────────────────────────┬──────────────┘       └─────────────────────────┬──────────────────────────────┘
                                          │                                                │
                                          └───────────────────────┬────────────────────────┘
                                                                  │
                                                                  ▼
                                  ┌────────────────────────────────────────────────────────┐
                                  │             Chromium (Headless / Visible)              │
                                  │                 https://chatgpt.com                    │
                                  └────────────────────────────────────────────────────────┘
```

---

## 3. Chrome DevTools Protocol (CDP) & Stealth Automation

Rather than running an external bridge like ChromeDriver, `chatgpt-spider` uses `github.com/go-rod/rod` to establish a direct WebSocket connection to Chromium's native debugging port.

### Anti-Bot Stealth Injection
Cloudflare Turnstile and OpenAI inspect hundreds of browser fingerprints. `chatgpt-spider` applies `github.com/go-rod/stealth` to every page prior to navigation:
* **Hides `navigator.webdriver`**: Overrides the flag so JavaScript reports `false`.
* **Mocks Hardware Concurrency & Memory**: Injects realistic values for `navigator.hardwareConcurrency` and `navigator.deviceMemory`.
* **Emulates Real Chrome Plugins & MimeTypes**: Populates fake plugin arrays mimicking a standard desktop Chrome installation.
* **Canvas & WebGL Evasion**: Normalizes WebGL rendering contexts to prevent hardware fingerprinting flags.
* **Humanized Key Actions**: Types text through native Chromium input events rather than setting raw `.value` attributes, triggering React's synthetic event dispatcher.

---

## 4. The Core Spider Engine (`internal/spider`)

The core engine encapsulates the entire automation lifecycle into a clean, reusable Go package:

```go
type Engine struct {
    mu          sync.Mutex        // Serializes browser actions on the active page
    convMu      sync.RWMutex      // Non-blocking read lock for active conversation ID
    inst        *browser.BrowserInstance
    currentConv string
}
```

### Key Responsibilities
1. **Session Lifecycle**: Initializes the browser, restores cookies from the persistent profile, and opens ChatGPT.
2. **Thread Navigation**: Navigates cleanly between conversation threads using `https://chatgpt.com/c/<uuid>` or triggers a fresh chat via `NewChat()`.
3. **Atomic Execution**: Ensures only one prompt is processed at a time on the underlying browser tab, avoiding race conditions where two simultaneous requests type into the prompt box together.
4. **Token Streaming Channel**: Accepts an `OnToken(delta string)` callback in `PromptRequest`, streaming tokens to the caller as they appear in the browser.

---

## 5. Real-Time Token Streaming Mechanics (`internal/conversation`)

One of the most complex challenges in web scraping is extracting real-time streaming tokens without network-layer interception (which gets blocked by Cloudflare). `chatgpt-spider` solves this with **Turn-Aware DOM Token Streaming**.

### The Multi-Turn Dilemma
In an ongoing chat thread with 5 previous messages:
1. If the scraper checks for `.markdown` or `[data-message-author-role='assistant']`, the 5th message already exists.
2. If the scraper begins reading immediately, it will mistakenly read the previous turn's message, emit it as a delta, and finish before ChatGPT even starts typing the new answer!

### The 6-Step Streaming Algorithm (`StreamCompletion`)

```
 [Step 1: Baseline]
 CountAssistantMessages() -> initialCount = N (e.g. 5)
         │
         ▼
 [Step 2: Prompt Dispatch]
 SendPrompt() -> Focus #prompt-textarea -> Insert Text -> Click Send
         │
         ▼
 [Step 3: Wait for Turn Initiation]
 Loop until:
   a) Stop button appears ("Stop streaming")  OR
   b) len(assistantElements) > initialCount (N+1)
         │
         ▼
 [Step 4: Real-Time Delta Emission]
 Target new element: assistantElements[N]
 Loop while generating:
   currentText = assistantElement.Text()
   if len(currentText) > len(lastObservedText):
       delta = currentText[len(lastObservedText):]
       onToken(delta)  // Emits live token chunk
       lastObservedText = currentText
         │
         ▼
 [Step 5: Termination Detection]
 Stop button disappears AND currentText stabilizes across 3 polls (~360ms)
         │
         ▼
 [Step 6: ID Resolution]
 Read updated page URL: /c/<conversation-id>
 Return full text & UUID
```

### Why DOM Streaming Over Network Hijacking?
In earlier designs, an HTTP interceptor attempted to hijack `/backend-api/conversation` via Go's `http.Client`. However:
* Go's default HTTP client lacks Cloudflare clearance tokens and browser TLS fingerprints, leading to `403 Forbidden` errors.
* Hijacking in Chromium pauses the request; if Go fails to forward it properly, the browser hangs indefinitely.
* In contrast, **DOM Streaming** allows Chromium's native HTTP/2 networking to handle all TLS and Cloudflare negotiation legitimately. The scraper simply reads the visible text as the browser renders it.

### Long Text Handling & Pasted Document Uploads
When a user (or automated script) inputs very large text into ChatGPT (e.g., long code files, JSON payloads, or multi-page documents):
1. **ChatGPT Automatic Card Conversion**: ChatGPT's React input handler intercepts large inputs and automatically converts the raw text into a virtual document attachment pill (showing `{document_id... Pasted text`).
2. **Asynchronous Upload State**: While creating this attachment, ChatGPT uploads the text payload in the background. During this processing period:
   * The Send button is set to `disabled=""` and `aria-disabled="true"`.
   * Pressing the Enter key does nothing because form submission is gated by the upload state.
3. **The Solution in `chatgpt-spider`**:
   In `SendPrompt`, after injecting the prompt, the engine enters a resilient polling loop:
   ```go
   // Wait up to 30s for pasted text upload / document attachment to finish
   for time.Now().Before(deadline) {
       for _, btnSel := range sendButtonSelectors {
           btn, _ := page.Timeout(300 * time.Millisecond).Element(btnSel)
           if btn != nil {
               disabled, _ := btn.Attribute("disabled")
               ariaDisabled, _ := btn.Attribute("aria-disabled")
               if disabled == nil && (ariaDisabled == nil || *ariaDisabled != "true") {
                   return btn.Click(proto.InputMouseButtonLeft, 1)
               }
           }
       }
       time.Sleep(250 * time.Millisecond)
   }
   ```
   If the prompt is short, the button is immediately enabled and clicked in <300ms. If the prompt is large and triggers a "Pasted text" document upload, the engine safely waits until the upload finishes and the button becomes active before dispatching.

---

## 6. Concurrency Architecture & Deadlock Prevention

### The Problem: Mutex Re-entrancy
In Go, `sync.Mutex` is **not re-entrant**. If Goroutine A holds `mu.Lock()` and attempts to call another method on the same struct that also executes `mu.Lock()`, Goroutine A will deadlock with itself forever.

During live SSE streaming:
1. `Prompt()` acquires `e.mu.Lock()`.
2. `StreamCompletion()` spawns a goroutine to read tokens and invokes `OnToken(delta)`.
3. `OnToken` builds the OpenAI `chat.completion.chunk` SSE payload, which needs `ConversationID`.
4. If `GetCurrentConversationID()` also locks `e.mu`, **DEADLOCK OCCURS IMMEDIATELY**.

### The Solution: Decoupled Concurrency (`convMu sync.RWMutex`)
`chatgpt-spider` uses two separate synchronization primitives:
1. **`e.mu sync.Mutex`**: Locks the browser page actions (typing, clicking, waiting for completion).
2. **`e.convMu sync.RWMutex`**: An independent, lightweight read-write mutex strictly protecting `currentConv`.

```go
// GetCurrentConversationID is non-blocking and safe to call concurrently during streaming
func (e *Engine) GetCurrentConversationID() string {
    e.convMu.RLock()
    id := e.currentConv
    e.convMu.RUnlock()
    if id != "" {
        return id
    }
    if e.inst != nil && e.inst.Page != nil {
        return conversation.GetCurrentConversationID(e.inst.Page)
    }
    return ""
}
```

This guarantees that token chunks can query the conversation ID hundreds of times per second during generation with zero lock contention.

---

## 7. Process Lifecycle & Windows Lockfile Resilience

### Windows Sharing Violations (Error Code 32)
When Chromium launches with `--user-data-dir`, it creates a singleton lockfile (`Lockfile`, `SingletonLock`) to prevent concurrent instances from corrupting the profile directory.
If the automation process crashes or is terminated with `Ctrl+C`:
1. The parent Go process exits.
2. Background Chromium helper processes (`crashpad-handler`, `gpu-process`) may linger for several seconds holding an open file handle.
3. Subsequent launches encounter:
   `Lock file can not be created! Error code: 32 (ERROR_SHARING_VIOLATION)`

### Multi-Tier Defense:
1. **Pre-Launch Sanitization**: Before starting Chromium, `browser.Launch()` aggressively removes stale lockfiles:
   ```go
   _ = os.Remove(filepath.Join(userDataDir, "lockfile"))
   _ = os.Remove(filepath.Join(userDataDir, "Lockfile"))
   _ = os.Remove(filepath.Join(userDataDir, "SingletonLock"))
   _ = os.Remove(filepath.Join(userDataDir, "SingletonSocket"))
   _ = os.Remove(filepath.Join(userDataDir, "SingletonCookie"))
   _ = os.Remove(filepath.Join(userDataDir, "Default", "lockfile"))
   ```
2. **Signal Traps (`SIGINT` / `SIGTERM`)**: Both the CLI engine and the REST server capture termination signals and synchronously invoke `launcher.Kill()`, terminating the entire child process tree.

---

## 8. OpenAI REST API Server (`internal/server`)

The `serve` command spins up a native Go HTTP server exposing endpoints that strictly match the OpenAI API specification:

### 1. `POST /v1/chat/completions` (Streaming Mode)
When `"stream": true` is passed:
* Sets headers:
  * `Content-Type: text/event-stream`
  * `Cache-Control: no-cache`
  * `Connection: keep-alive`
* Immediately flushes the assistant role chunk:
  ```json
  data: {"id":"chatcmpl-...","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant"}}]}
  ```
* As tokens arrive from `StreamCompletion`, flushes content chunks:
  ```json
  data: {"id":"chatcmpl-...","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"content":"Hello"}}]}
  ```
* When generation ends, flushes the stop reason and termination marker:
  ```text
  data: {"id":"chatcmpl-...","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

  data: [DONE]
  ```

### 2. Multi-Turn Thread Continuity
Clients can maintain conversation threads across requests by passing `conversation_id`:
```json
{
  "conversation_id": "6a9eec31-5bac-83e9-985a-ee041b8b3247",
  "messages": [
    {"role": "user", "content": "What did I ask you previously?"}
  ]
}
```
The server checks if the requested conversation is already open; if not, it navigates to that thread before sending the prompt.

### 3. `GET /v1/history`
Scrapes and returns the visible conversation turns of the active thread:
```json
{
  "conversation_id": "6a9eec31-5bac-83e9-985a-ee041b8b3247",
  "messages": [
    {"role": "user", "content": "Hello"},
    {"role": "assistant", "content": "Hi there! How can I help you today?"}
  ]
}
```

---

## 9. Session Persistence & Credential Security (`internal/auth`)

### Storage Hierarchy
1. **Browser Profile Directory**: `~/.chatgpt-spider/profile`
   Contains browser cookies, LocalStorage, and indexedDB data. Logging in once visually (`--headless=false`) keeps you logged in across future runs.
2. **Encrypted Token Vault**: `~/.chatgpt-spider/credentials.json`
   Stores session cookies (`__Secure-next-auth.session-token`) encrypted with **AES-256 GCM**.
   * The encryption key is derived using SHA-256 on machine-specific hardware attributes (`hostname`).
   * Handles chunked session tokens (`.0` and `.1`) automatically.

---

## 10. Embedding as a Go Library

Because `internal/spider` has no global state or CLI dependencies, you can import it into custom Go applications:

```go
package main

import (
    "context"
    "fmt"
    "chatgpt-spider/internal/spider"
)

func main() {
    // 1. Create engine
    eng, err := spider.NewEngine(spider.Options{
        Headless: true,
    })
    if err != nil {
        panic(err)
    }
    defer eng.Close()

    // 2. Open ChatGPT
    if err := eng.Initialize(""); err != nil {
        panic(err)
    }

    // 3. Prompt with typewriter token streaming
    resp, err := eng.Prompt(context.Background(), spider.PromptRequest{
        Prompt: "Explain concurrency in Go in 2 sentences.",
        OnToken: func(token string) {
            fmt.Print(token)
        },
    })
    if err != nil {
        panic(err)
    }

    fmt.Printf("\nDone! Conversation UUID: %s\n", resp.ConversationID)
}
```
