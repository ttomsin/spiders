# What You Can Use chatgpt-spider For 🕷️

A guide to real-world applications, capabilities, and technical boundaries of **`chatgpt-spider`**—the native Go automation engine and OpenAI-compatible REST server for ChatGPT.

---

## Table of Contents
- [1. Executive Summary](#1-executive-summary)
- [2. Prime Use Cases (What It Excels At)](#2-prime-use-cases-what-it-excels-at)
  - [A. Drop-in OpenAI API Replacement for Developer Tools](#a-drop-in-openai-api-replacement-for-developer-tools)
  - [B. Multi-Megabyte Data & Document Analysis (Attachment Simulation)](#b-multi-megabyte-data--document-analysis-attachment-simulation)
  - [C. Strict Structured Output Extraction (`--format`)](#c-strict-structured-output-extraction---format)
  - [D. CLI Power Tools, Scripts & CI/CD Pipelines](#d-cli-power-tools-scripts--cicd-pipelines)
  - [E. Continuous Multi-Turn Agentic Workflows](#e-continuous-multi-turn-agentic-workflows)
  - [F. Webhook-Driven Background Automation & Batch Jobs](#f-webhook-driven-background-automation--batch-jobs)
- [3. What Might Look Impossible (And How chatgpt-spider Solves It)](#3-what-might-look-impossible-and-how-chatgpt-spider-solves-it)
  - [1. "Cloudflare / Bot Defense Blocks Every Headless Browser"](#1-cloudflare--bot-defense-blocks-every-headless-browser)
  - [2. "Pasting a 3MB / 50,000-Line File Hangs or Crashes Browser Inputs"](#2-pasting-a-3mb--50000-line-file-hangs-or-crashes-browser-inputs)
  - [3. "Real-Time Streaming is Impossible Without Reverse-Engineering Private Endpoints"](#3-real-time-streaming-is-impossible-without-reverse-engineering-private-endpoints)
  - [4. "LLMs on Free/Web UIs Never Adhere Strictly to Clean JSON"](#4-llms-on-freeweb-uis-never-adhere-strictly-to-clean-json)
  - [5. "Chromium Orphan Processes and Profile Lockfile Collisions on Windows"](#5-chromium-orphan-processes-and-profile-lockfile-collisions-on-windows)
  - [6. "Web UIs Change Class Names Every Few Weeks"](#6-web-uis-change-class-names-every-few-weeks)
- [4. Genuine Constraints & Hard Limits (What Truly Isn't Possible Today)](#4-genuine-constraints--hard-limits-what-truly-isnt-possible-today)
  - [A. Concurrency per Browser Instance](#a-concurrency-per-browser-instance)
  - [B. Sub-Second Latency / High-Throughput Production APIs](#b-sub-second-latency--high-throughput-production-apis)
  - [C. Automated 2FA / CAPTCHA Solving in Pure Headless Anonymous Mode](#c-automated-2fa--captcha-solving-in-pure-headless-anonymous-mode)
  - [D. Multi-Tab Simultaneous Generation in a Single Profile](#d-multi-tab-simultaneous-generation-in-a-single-profile)
- [5. Practical Workflow Playbook](#5-practical-workflow-playbook)
  - [Playbook 1: Coding Copilot with Cursor / VS Code](#playbook-1-coding-copilot-with-cursor--vs-code)
  - [Playbook 2: Automated JSONL / CSV Data Pipeline](#playbook-2-automated-jsonl--csv-data-pipeline)
  - [Playbook 3: Resumable Long-Form Research Assistant](#playbook-3-resumable-long-form-research-assistant)

---

## 1. Executive Summary

`chatgpt-spider` bridges the gap between the rich, multimodal, file-handling web interface of ChatGPT and programmatic developer tooling. Instead of paying per-token API charges or struggling with broken webdriver dependencies, `chatgpt-spider` runs a headless stealth Chromium instance over native Chrome DevTools Protocol (CDP) and surfaces it as:
1. An **OpenAI-compatible REST API** (`http://localhost:8080/v1/chat/completions`) with real-time SSE token streaming.
2. A **scriptable CLI tool** with file attachment support (`--file`), format enforcement (`--format`), and export options.
3. An **embeddable Go library** (`internal/spider`) for microservices and background daemons.

---

## 2. Prime Use Cases (What It Excels At)

### A. Drop-in OpenAI API Replacement for Developer Tools
Expose ChatGPT directly to developer environments without modifying their code:
- **Cursor IDE / VS Code Continue**: Configure OpenAI base URL to `http://localhost:8080/v1` and API key to `any-key`. You get free, high-context code completion powered by ChatGPT.
- **Python / LangChain / LlamaIndex**: Point `ChatOpenAI(base_url="http://localhost:8080/v1")` directly at your local spider.
- **AutoGPT / Local Agents**: Route tool calls and agent loops through the local REST daemon.

### B. Multi-Megabyte Data & Document Analysis (Attachment Simulation)
Most API gateways reject payloads over 100KB–500KB or charge exorbitant token fees. ChatGPT's web UI accepts massive files by compiling pasted text into document attachments.
- **Dataset Analysis**: Pass 3MB+ JSONL logs, 50,000-line server traces, or raw CSV exports using `--file`:
  ```powershell
  .\chatgpt-spider.exe prompt "Find the top error spikes and outliers" --file "D:\logs\production_traffic.jsonl" --format json
  ```
- **Codebase Auditing**: Dump an entire codebase or AST dump into a single file and feed it to the model for architecture audits.

### C. Strict Structured Output Extraction (`--format`)
When orchestrating automated pipelines, conversational chit-chat ("*Sure, I'd be happy to help with that!*") breaks JSON parsers.
- `--format json`: Guarantees raw, valid JSON ready for `jq`, PowerShell `ConvertFrom-Json`, or automated database insertion.
- `--format csv`: Outputs valid tabular data with column headers.
- `--format xml` / `--format yaml`: Standard interchange formats.
- **Custom Schema Constraints**: Enforce exact keys and types:
  ```powershell
  .\chatgpt-spider.exe prompt "Extract flight details from this ticket" --file "ticket.txt" --format '{"flight_no": string, "departure": string, "gate": string}'
  ```

### D. CLI Power Tools, Scripts & CI/CD Pipelines
- **Pre-commit PR Summarizer**: Run a git hook that feeds `git diff` into `chatgpt-spider prompt --format markdown` to generate pull request descriptions.
- **Automated Bug Triage**: Send stack traces from your CI runner into ChatGPT and output recommended fixes to a markdown file (`-o bug_report.md`).
- **Terminal Shell Assistant**: Run quick queries from PowerShell or Bash with immediate typewriter streaming.

### E. Continuous Multi-Turn Agentic Workflows
The engine captures and exposes the conversation UUID from ChatGPT's URL.
- Thread continuation across calls:
  ```powershell
  # First turn
  .\chatgpt-spider.exe prompt "Analyze this schema" --file "schema.sql"
  # Continue in same thread
  .\chatgpt-spider.exe prompt "Write migrations for SQLite" --chat-session-id <UUID>
  ```
- History inspection: Retrieve past turns programmatically via `--history` or `GET /v1/history`.

### F. Ephemeral Single-Turn Workflows (`--session-delete`)
When running automated scripts, testing code snippets, or querying confidential data:
- Pass `--session-delete` to immediately wipe the conversation thread from your ChatGPT account sidebar the second the response completes.
- Keeps your ChatGPT account sidebar 100% clean and uncluttered.
- Supported in the interactive REPL (`/delete`), CLI (`--session-delete`), and REST server (`"session_delete": true`).

### G. Webhook-Driven Background Automation & Batch Jobs
- **Batch Processing**: Run a file containing 100 prompts in sequence with `-b prompts.txt -o results.json` with rate-limiting pauses.
- **Webhook Dispatch**: Automatically dispatch responses to Slack, Discord, or an internal webhook endpoint upon completion (`--webhook https://my-webhook.com`).

---

## 3. What Might Look Impossible (And How chatgpt-spider Solves It)

There is a common belief that certain operations cannot be done with ChatGPT automation without reverse-engineering encrypted APIs or triggering bans. Here is how `chatgpt-spider` turns those "impossible" problems into solved capabilities:

---

### 1. "Cloudflare / Bot Defense Blocks Every Headless Browser"
* **The Common Belief**: Launching Chrome with `--headless` immediately trips Cloudflare Turnstile bot challenges because `navigator.webdriver` is true, user-agent flags give it away, and CDP automation markers are exposed.
* **Why It Works in chatgpt-spider**:
  - Uses `go-rod/stealth` to evaluate evasion scripts before any document scripts execute.
  - Masks `navigator.webdriver = undefined`, mocks authentic Chrome plugins and mimeTypes, normalizes WebGL renderer vendor strings, and randomizes viewport dimensions.
  - Operates using authenticated session cookies saved in a persistent user data directory (`~/.chatgpt-spider/profile`), meaning Cloudflare sees an already-authenticated, recognized browser profile.

---

### 2. "Pasting a 3MB / 50,000-Line File Hangs or Crashes Browser Inputs"
* **The Common Belief**: Sending 3MB of text into a browser `textarea` or `contenteditable` div freezes the browser tab or gets truncated.
* **Why It Works in chatgpt-spider**:
  - Emulates a synthetic `ClipboardEvent('paste')` with a custom `DataTransfer` object.
  - ChatGPT's client-side listener intercepts large clipboard events and triggers its native **Pasted Text Attachment Card** (`{document_id... Pasted text}`).
  - The actual user instruction (e.g. *"Analyze this data and output JSON"*) is inserted cleanly *below* the document card into the composer textarea via `document.execCommand('insertText')`.
  - The engine dynamically monitors upload spinners (`[role='progressbar']`, radial progress SVGs, and upload error alerts) and waits until the attachment is fully parsed before clicking Send.

---

### 3. "Real-Time Streaming is Impossible Without Reverse-Engineering Private Endpoints"
* **The Common Belief**: To get true streaming (character-by-character or chunk-by-chunk), you must reverse-engineer OpenAI's private `backend-anon/conversation` or WebSocket protocol, complete with Proof-of-Work (PoW) headers and device integrity tokens.
* **Why It Works in chatgpt-spider**:
  - Uses **DOM-Native Token Slicing**: The spider watches the live DOM container of the latest assistant `<article>`.
  - It maintains an index of `lastObservedText`. Whenever ChatGPT renders new tokens, the engine calculates:
    Delta = currentText[len(lastObservedText):]
  - That delta is immediately flushed over stdout (in CLI mode) or as an SSE chunk (`data: {"choices":[{"delta":{"content":"..."}}]}`) over HTTP.
  - Zero private endpoint reverse engineering; 100% resilient to network-level encryption or authentication changes.

---

### 4. "LLMs on Free/Web UIs Never Adhere Strictly to Clean JSON"
* **The Common Belief**: Without OpenAI's official `response_format: {"type": "json_object"}` API parameter, the web model always adds conversational preamble: *"Sure! Here is the JSON you requested..."* followed by markdown code fences.
* **Why It Works in chatgpt-spider**:
  - `internal/format` applies programmatic system directive wrappers at the prompt level.
  - Instructs the model to output **strictly raw, unescaped, valid syntax** with zero markdown backticks and zero intro/outro remarks.
  - For HTTP REST clients, passing `"response_format": {"type": "json_object"}` or `"format": "json"` automatically applies these prompt constraints transparently.

---

### 5. "Chromium Orphan Processes and Profile Lockfile Collisions on Windows"
* **The Common Belief**: Running automated browsers in scripts or web servers on Windows leads to lingering zombie `chrome.exe` processes that lock the profile directory (`SingletonLock`), preventing future runs from starting.
* **Why It Works in chatgpt-spider**:
  - Before launching, the engine checks for and removes stale lockfiles (`Lockfile`, `SingletonLock`, `SingletonCookie`, `SingletonSocket`).
  - Registers OS signal listeners (`SIGINT`, `SIGTERM`) to trigger graceful teardown of browser contexts.
  - Coordinates thread-safe browser access using decoupled read-write mutexes (`sync.RWMutex` for conversation state, `sync.Mutex` for page dispatch), preventing multi-threaded crashes.

---

### 6. "Web UIs Change Class Names Every Few Weeks"
* **The Common Belief**: Web automation breaks every time OpenAI pushes an update with minified CSS classes (`.css-1f4a9b`).
* **Why It Works in chatgpt-spider**:
  - Does **not** rely on ephemeral CSS classes.
  - Uses structural and semantic selectors that OpenAI preserves for accessibility and DOM hierarchy:
    - Input: `#prompt-textarea`, `div[contenteditable='true']`
    - Send button: `button[data-testid='send-button']`, `button[aria-label='Send prompt']`
    - Stop button: `button[data-testid='stop-button']`, `button[aria-label='Stop streaming']`
    - Message containers: `article[data-message-author-role='assistant']`, `.markdown`

---

## 4. Genuine Constraints & Hard Limits (What Truly Isn't Possible Today)

While `chatgpt-spider` achieves things that seem impossible for browser scrapers, honesty about actual technical boundaries is vital for production system design:

### A. Concurrency per Browser Instance
* **Limit**: Single-turn concurrency per browser profile.
* **Why**: A single browser tab on ChatGPT cannot generate two responses simultaneously in the same conversation.
* **Workaround**: For multiple simultaneous requests, multiple `chatgpt-spider` instances can be launched with separate user profile directories and different ports (e.g., ports 8080, 8081, 8082).

### B. Sub-Second Latency / High-Throughput Production APIs
* **Limit**: Response start latency is typically **1.5s to 3.5s** (browser DOM initialization + typing simulation + server generation time).
* **Comparison**: Direct official API endpoints have 200ms–800ms Time-To-First-Token (TTFT).
* **Verdict**: `chatgpt-spider` is ideal for developer tooling, offline batch jobs, long-context data processing, and copilot environments, but is not designed for 100+ requests-per-second real-time consumer web applications.

### C. Automated 2FA / CAPTCHA Solving in Pure Headless Anonymous Mode
* **Limit**: If OpenAI forces an interactive puzzle captcha (e.g., Arkose Labs rotating animals) during an unauthenticated guest session, the headless engine cannot solve it autonomously without a 3rd-party CAPTCHA solver.
* **Workaround**: Run with `--headless=false` once to log in manually. Your authenticated cookies and session token will be saved to `~/.chatgpt-spider/profile`. After that, you can run completely headless without seeing captchas.

### D. Multi-Tab Simultaneous Generation in a Single Profile
* **Limit**: Attempting to open 10 tabs under the same user profile to prompt in parallel triggers OpenAI account-level rate limits and tab sync conflicts.
* **Verdict**: Serialize requests or queue them using the built-in batch command (`chatgpt-spider batch -f prompts.txt`).

---

## 5. Practical Workflow Playbook

### Playbook 1: Coding Copilot with Cursor / VS Code
1. Start the spider server:
   ```powershell
   .\chatgpt-spider.exe serve --port 8080
   ```
2. Open **Cursor Settings** > **Models** > **OpenAI API Key**:
   - Base URL: `http://localhost:8080/v1`
   - API Key: `dummy-key`
   - Model Name: `gpt-4o`
3. Enjoy full AI code editing and chat streaming directly through ChatGPT without per-token charges.

---

### Playbook 2: Automated JSONL / CSV Data Pipeline
Analyze massive datasets and ingest clean structured JSON directly into a database or analysis script:
```powershell
.\chatgpt-spider.exe prompt "Analyze this customer support dump. Group issues into categories with counts and severity levels." `
  --file "D:\data\customer_tickets.jsonl" `
  --format json `
  --output "D:\data\ticket_summary.json"
```

---

### Playbook 3: Resumable Long-Form Research Assistant
1. Initialize the thread:
   ```powershell
   .\chatgpt-spider.exe prompt "You are a senior financial analyst. I will feed you quarterly reports."
   # Output displays: Session ID: 6a9ef469-6be0-83e9-9d96-5be42631947b
   ```
2. Feed Q1 earnings report:
   ```powershell
   .\chatgpt-spider.exe prompt "Here is the Q1 earnings report. Extract EBITDA and revenue growth." `
     --file "Q1_report.txt" `
     --chat-session-id 6a9ef469-6be0-83e9-9d96-5be42631947b
   ```
3. Feed Q2 earnings report and compare:
   ```powershell
   .\chatgpt-spider.exe prompt "Here is Q2. Compare the operating margins against Q1." `
     --file "Q2_report.txt" `
     --chat-session-id 6a9ef469-6be0-83e9-9d96-5be42631947b `
     --format markdown `
     --output "comparison.md"
   ```

---

## Summary Matrix

| Feature / Need | Legacy Webdrivers | Direct OpenAI API | `chatgpt-spider` 🕷️ |
|---|---|---|---|
| **External Driver Setup** | Required (`chromedriver`) | None | **Zero (Native CDP)** |
| **Token Cost** | $0 | Per Token | **$0** |
| **Real-Time Token Streaming** | ❌ (Full wait) | ✅ (SSE) | **✅ (Live DOM SSE + CLI)** |
| **Large File / Text Ingestion** | ⚠️ (Input freeze) | ⚠️ (Context/payload limits) | **✅ (Native Document Cards)** |
| **Strict Output Formatting** | ❌ | ✅ (`json_schema`) | **✅ (`--format` engine)** |
| **OpenAI Drop-In Compatibility** | ❌ | Native | **✅ (`/v1/chat/completions`)** |
| **Stealth & Anti-Bot Masking** | ❌ (Detected) | N/A | **✅ (Pre-nav stealth)** |
| **Sub-Second Throughput** | ❌ | ✅ | ⚠️ (Browser speed) |
