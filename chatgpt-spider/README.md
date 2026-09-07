# chatgpt-spider 🕷️

```text
      / _ \                  _           _              _     ____        _     _           
    \_\(_)/_/           ____| |__   __ _| |_ __ _ _ __ | |_   / ___| _ __ (_) __| | ___ _ __ 
     _//o\\_            / __/| '_ \ / _` | __/ _` | '_ \| __|  \___ \| '_ \| |/ _` |/ _ \ '__|
      /   \             | (__ | | | | (_| | || (_| | |_) | |_    ___) | |_) | | (_| |  __/ |   
     /     \            \___|_| |_|\__,_|\__\__, | .__/ \__|  |____/| .__/|_|\__,_|\___|_|   
                                              |___/|_|                |_|                      
                                   [Native ChatGPT Automation Engine & REST Server • v1.0.0]
```

**chatgpt-spider** is a high-performance, modular ChatGPT automation engine and OpenAI-compatible REST server written in Go. It bypasses the need for Selenium or external webdriver binaries by driving Chromium directly via the **Chrome DevTools Protocol (CDP)** with automated stealth evasion, real-time token streaming, and profile persistence.

---

## Table of Contents
- [Key Features](#key-features)
- [How chatgpt-spider Works](#how-chatgpt-spider-works)
- [OpenAI-Compatible REST Server](#openai-compatible-rest-server)
  - [Endpoints](#endpoints)
  - [Live Streaming with curl (PowerShell)](#live-streaming-with-curl-powershell)
  - [Non-Streaming with Invoke-RestMethod](#non-streaming-with-invoke-restmethod)
  - [Integration with Python / OpenAI SDK](#integration-with-python--openai-sdk)
  - [Integration with Cursor / LangChain](#integration-with-cursor--langchain)
- [CLI Command Reference](#cli-command-reference)
  - [Interactive Terminal Chat (REPL)](#1-interactive-terminal-chat-repl)
  - [Single Prompt](#2-single-prompt)
  - [Fetch In-Conversation History](#3-fetch-in-conversation-history)
  - [Batch Processing from File](#4-batch-processing-from-file)
- [Using as a Go Library (`internal/spider`)](#using-as-a-go-library)
- [Authentication & Profiles](#authentication--profiles)
- [License](#license)

---

## Key Features

- 🚀 **Zero External Webdrivers**: No `chromedriver.exe` or Selenium grid required. Communicates directly over native Chromium DevTools Protocol (CDP).
- ⚡ **Real-Time Token Streaming**: True typewriter streaming effect in CLI and Server-Sent Events (SSE) for HTTP clients.
- 🌐 **OpenAI-Compatible REST API**: Drop-in replacement for `POST /v1/chat/completions`, `GET /v1/models`, and `GET /v1/history`.
- 🧩 **Modular Core Engine**: Clean Go package (`internal/spider`) separating the browser orchestration logic from CLI commands and HTTP servers.
- 📜 **In-Conversation History**: Extract and inspect visible message turns from any active or historical conversation thread (`--history` / `/history` / `GET /v1/history`).
- 🤖 **Headless by Default**: Runs silently in the background with minimal footprint.
- 🛡️ **Anti-Bot Stealth**: Injects evasive navigator overrides, hides automation flags (`navigator.webdriver`), and simulates human-like interaction timings.
- 🔐 **Persistent Profiles & Guest Mode**: Log in once to save session cookies in `~/.chatgpt-spider/profile`, or run pure guest sessions with `--anon`.
- 🔄 **Safe Concurrency & Process Cleanup**: Dedicated read-write locks (`sync.RWMutex`) prevent deadlocks, and automatic lockfile sanitizers prevent Chromium sharing collisions.

---

## How chatgpt-spider Works

> 📖 **Deep Technical Dive**: For an exhaustive architectural breakdown including token streaming algorithms, lockfile mechanics, and concurrency design, read [HOW-CHATGPT-SPIDER-WORKS.md](HOW-CHATGPT-SPIDER-WORKS.md).
> 
> 💡 **Use Cases & Capabilities Guide**: To explore real-world workflows, prime use cases, and what might look impossible vs true constraints, read [WHAT-YOU-CAN-USE-CHATGPT-SPIDER.md](WHAT-YOU-CAN-USE-CHATGPT-SPIDER.md).

```
                     ┌────────────────────────────────────────────────────────┐
                     │                     Your Application                   │
                     │   (CLI Tool / Cursor / LangChain / Python OpenAI SDK)  │
                     └───────────────────────────┬────────────────────────────┘
                                                 │ HTTP / CLI
                                                 ▼
                     ┌────────────────────────────────────────────────────────┐
                     │                 chatgpt-spider Engine                  │
                     │                   (internal/spider)                    │
                     ├───────────────────────────┬────────────────────────────┤
                     │  OpenAI REST Server       │  Interactive CLI REPL      │
                     │  (internal/server)        │  (cmd/chat, cmd/prompt)    │
                     └───────────────────────────┴────────────────────────────┘
                                                 │
                                                 ▼
                     ┌────────────────────────────────────────────────────────┐
                     │          Native Chrome DevTools Protocol (CDP)         │
                     │                (go-rod + stealth engine)               │
                     └───────────────────────────┬────────────────────────────┘
                                                 │
                        ┌────────────────────────┴────────────────────────┐
                        ▼                                                 ▼
             ┌─────────────────────┐                           ┌─────────────────────┐
             │  ChatGPT Web Page   │                           │ Persistent Storage  │
             │   (chatgpt.com)     │                           │ (~/.chatgpt-spider) │
             ├─────────────────────┤                           ├─────────────────────┤
             │ • Prompt Input Area │                           │ • Browser Profile   │
             │ • Stop/Gen Controls │                           │ • Encrypted Tokens  │
             │ • Turn Message DOM  │                           │ • Lockfile Cleaner  │
             └─────────────────────┘                           └─────────────────────┘
```

### 1. The Modular Core (`internal/spider/engine.go`)
At the center of `chatgpt-spider` is the `Engine` struct. It wraps:
- A stealth browser instance managed by `internal/browser`.
- Concurrency controls: a primary `sync.Mutex` serializing prompt dispatch on the single active browser tab, and an independent `sync.RWMutex` (`convMu`) ensuring non-blocking reads of the active `conversation_id`.
- Thread lifecycle controls: initialization, thread switching (`OpenConversation`), starting fresh chats (`NewChat`), and reading visible turn history (`GetHistory`).

### 2. Turn-Aware Real-Time Token Streaming (`StreamCompletion`)
Instead of relying on fragile network proxies or slow polling, `chatgpt-spider` uses a stateful DOM tracking algorithm:
1. **Pre-Prompt Baseline Count**: Before sending the prompt, `CountAssistantMessages(page)` counts the exact number of assistant `<article>` elements already present in the DOM.
2. **Prompt Injection & Transmission**: The engine focuses `#prompt-textarea`, inserts text using native input events, and triggers submission via the Send button or Enter key.
3. **Generation Detection**: The engine monitors the ChatGPT UI for the appearance of the Stop Generation button (`button[data-testid='stop-button']`).
4. **Targeted Message Isolation**: The polling loop waits until `len(assistantMsgs) > initialCount`. This guarantees that the engine **never** reads or leaks text from the previous turn.
5. **Delta Calculation & Flush**: As tokens are typed onto the page, the engine slices new text `delta = text[len(lastObservedText):]` and immediately invokes the `OnToken(delta)` callback (triggering terminal typewriter output or flushing an SSE `chat.completion.chunk`).
6. **Completion Detection**: When the Stop button disappears and the message text stabilizes across consecutive polls, the engine finalizes the turn and captures the updated conversation UUID from the browser URL.

### 3. Concurrency & Deadlock Prevention
In Go, `sync.Mutex` is not re-entrant. `chatgpt-spider` decouples conversation metadata (`convMu sync.RWMutex`) from the browser automation lock (`mu sync.Mutex`). When streaming SSE chunks over HTTP, `OnToken` can safely resolve `GetCurrentConversationID()` without blocking or deadlocking with the ongoing generation goroutine.

### 4. Process & Lockfile Resilience
On Windows, abruptly killing browser processes can leave Chromium lockfiles (`Lockfile`, `SingletonLock`, `Default/lockfile`) on disk. `chatgpt-spider` automatically purges stale lockfiles before launch and registers OS signal traps (`SIGINT`, `SIGTERM`) to ensure `launcher.Kill()` terminates all child helper processes cleanly.

---

## OpenAI-Compatible REST Server

Start the server on any port (default: 8080):
```powershell
.\chatgpt-spider.exe serve --port 8080
```

### Endpoints

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/v1/chat/completions` | OpenAI-compatible chat completion. Supports `"stream": true` and `"stream": false`. Accepts optional `conversation_id`. |
| `GET` | `/v1/models` | Returns available model list (`gpt-4o`, `chatgpt-spider`). |
| `GET` | `/v1/history` | Returns visible message turns of the active conversation thread. |
| `GET` | `/health` | Server healthcheck status. |

---

### Live Streaming with curl (PowerShell)

> **Note for Windows PowerShell**: In PowerShell, standard `curl` is an alias for `Invoke-WebRequest`. Use `curl.exe` with `--%` (stop-parsing) so JSON quotes are preserved verbatim:

```powershell
curl.exe --% -N http://localhost:8080/v1/chat/completions -H "Content-Type: application/json" -d "{\"stream\": true, \"messages\": [{\"role\": \"user\", \"content\": \"Count from 1 to 5\"}]}"
```

**Streamed Response**:
```text
data: {"id":"chatcmpl-1788800059247796100","object":"chat.completion.chunk","created":1788800059,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"chatcmpl-1788800059247796100","object":"chat.completion.chunk","created":1788800061,"model":"gpt-4o","conversation_id":"6a9eec31-5bac-83e9-985a-ee041b8b3247","choices":[{"index":0,"delta":{"content":"1, "},"finish_reason":null}]}

data: {"id":"chatcmpl-1788800059247796100","object":"chat.completion.chunk","created":1788800061,"model":"gpt-4o","conversation_id":"6a9eec31-5bac-83e9-985a-ee041b8b3247","choices":[{"index":0,"delta":{"content":"2, 3, "},"finish_reason":null}]}

data: {"id":"chatcmpl-1788800059247796100","object":"chat.completion.chunk","created":1788800062,"model":"gpt-4o","conversation_id":"6a9eec31-5bac-83e9-985a-ee041b8b3247","choices":[{"index":0,"delta":{"content":"4, 5"},"finish_reason":null}]}

data: {"id":"chatcmpl-1788800059247796100","object":"chat.completion.chunk","created":1788800062,"model":"gpt-4o","conversation_id":"6a9eec31-5bac-83e9-985a-ee041b8b3247","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

To continue the thread in the next request, pass the `conversation_id`:
```powershell
curl.exe --% -N http://localhost:8080/v1/chat/completions -H "Content-Type: application/json" -d "{\"stream\": true, \"conversation_id\": \"6a9eec31-5bac-83e9-985a-ee041b8b3247\", \"messages\": [{\"role\": \"user\", \"content\": \"What did you just count?\"}]}"
```

---

### Non-Streaming with Invoke-RestMethod

```powershell
$body = '{"stream": false, "messages": [{"role": "user", "content": "What is the capital of Japan?"}]}'
$response = Invoke-RestMethod -Uri "http://localhost:8080/v1/chat/completions" -Method Post -ContentType "application/json" -Body $body
$response.choices[0].message.content
# Output: The capital of Japan is Tokyo.
```

---

### Integration with Python / OpenAI SDK

You can drop `chatgpt-spider` directly into existing Python code using the official `openai` package:

```python
from openai import OpenAI

# Point client to your local chatgpt-spider instance
client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="not-needed"  # Authentication is handled by chatgpt-spider
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[
        {"role": "user", "content": "Explain Go channels in 2 sentences."}
    ],
    stream=True
)

for chunk in response:
    content = chunk.choices[0].delta.content or ""
    print(content, end="", flush=True)
print()
```

---

### Integration with Cursor / LangChain

- **Cursor**: In Settings > Models > OpenAI API Base, enter `http://localhost:8080/v1`.
- **LangChain**:
  ```python
  from langchain_openai import ChatOpenAI

  llm = ChatOpenAI(
      base_url="http://localhost:8080/v1",
      api_key="dummy",
      model="gpt-4o",
      streaming=True
  )
  ```

---

## CLI Command Reference

### 1. Interactive Terminal Chat (REPL)
```powershell
.\chatgpt-spider.exe chat
```
Inside the interactive REPL:
- Live typewriter streaming for every response.
- `/history`: Displays the full visible message history of the current thread.
- `/new`: Starts a fresh, clean conversation thread.
- `/open <uuid>`: Switches to a specific conversation by ID or URL.
- `exit` or `quit`: Exits cleanly.

### 2. Single Prompt
```powershell
# Send prompt and stream output to terminal
.\chatgpt-spider.exe prompt "Explain how mutexes work in Go"

# Save response to Markdown or JSON
.\chatgpt-spider.exe prompt "Summarize Kubernetes" -o kubernetes.md

# Forward response to an external webhook
.\chatgpt-spider.exe prompt "Generate weekly report" --webhook-url "https://api.yourdomain.com/ingest"
```

### 3. Fetch In-Conversation History
Inspect the turns of any conversation thread without sending a new message:
```powershell
.\chatgpt-spider.exe prompt --chat-session-id "6a9eec31-5bac-83e9-985a-ee041b8b3247" --history
```

### 4. Batch Processing from File

You can execute a list of prompts sequentially from a text file, with automatic delay between requests, and export the aggregated responses to JSON or Markdown.

#### 1. Create a `prompts.txt` file:
```text
# General knowledge prompts
Write a haiku about computers
Explain photosynthesis in two sentences
What is Moore's law?
What is the speed of sound?
```
*(Lines starting with `#` are treated as comments and ignored).*

#### 2. Run the batch command:
```powershell
# Output all results to JSON with a 3-second delay between prompts
.\chatgpt-spider.exe batch -i prompts.txt -o results.json --delay 3

# Or export to formatted Markdown:
.\chatgpt-spider.exe batch -i prompts.txt -o results.md -d 2

# Stream each completed prompt directly to an external webhook:
.\chatgpt-spider.exe batch -i prompts.txt --webhook-url "https://api.yourdomain.com/ingest"

# Run in anonymous / guest mode without logging in:
.\chatgpt-spider.exe batch -i prompts.txt -o results.json --anon
```

#### Batch Flags:
- `-i, --input <file>`: *(Required)* Path to the text file containing prompts (one per line).
- `-d, --delay <int>`: Delay in seconds between prompts to avoid rate limits (default: `3`).
- `-o, --output <file>`: Destination file (`.json` or `.md`).
- `--webhook-url <url>`: HTTP endpoint to POST each response turn as soon as it completes.
- `--anon`: Run in temporary guest mode without cookies.


---

## Using as a Go Library

Because the engine is fully decoupled into `internal/spider`, you can import and embed it directly into your own Go applications and microservices:

```go
package main

import (
	"context"
	"fmt"
	"chatgpt-spider/internal/spider"
)

func main() {
	// Initialize engine
	eng, err := spider.NewEngine(spider.Options{
		Headless: true,
	})
	if err != nil {
		panic(err)
	}
	defer eng.Close()

	_ = eng.Initialize("")

	// Send prompt with live token streaming callback
	resp, err := eng.Prompt(context.Background(), spider.PromptRequest{
		Prompt: "Hello! Tell me a fun fact about spiders.",
		OnToken: func(token string) {
			fmt.Print(token) // Live typewriter token output
		},
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("\nConversation ID: %s\n", resp.ConversationID)
}
```

---

## Authentication & Profiles

`chatgpt-spider` supports three authentication modes:

1. **Persistent Browser Session (Recommended)**:
   Launch once with `--headless=false` and log into your account:
   ```powershell
   .\chatgpt-spider.exe chat --headless=false
   ```
   Cookies and session state are persisted in `~/.chatgpt-spider/profile`. Subsequent runs can run headless (`--headless=true`) without logging in again.

2. **Anonymous Guest Mode**:
   Pass `--anon` to run against ChatGPT's public free tier in an ephemeral sandbox without cookies or saved sessions:
   ```powershell
   .\chatgpt-spider.exe prompt "What is the speed of light?" --anon
   ```

3. **Encrypted Session Token**:
   Store your `__Secure-next-auth.session-token` cookie with AES-256 GCM:
   ```powershell
   .\chatgpt-spider.exe auth set-token
   # Check status
   .\chatgpt-spider.exe auth status
   ```

---

## Global CLI Flags

```text
Flags:
      --anon                   Run in anonymous / guest mode without saved credentials
      --chat-session-id string Resume or inspect a specific conversation thread (UUID or URL)
      --debug                  Open browser devtools inspector
      --headless               Run browser in headless mode (default: true)
      --history                Fetch and display message history of the conversation
      --new-chat               Start a fresh conversation thread before sending prompt
  -o, --output string          Destination file path (.md or .json)
  -t, --session-token string   ChatGPT __Secure-next-auth.session-token cookie
      --webhook-url string     HTTP endpoint to receive conversation turns
```

---

## License
MIT License. Created for native, high-performance web automation.