# chatgpt-spider

**chatgpt-spider** is a high-performance native ChatGPT automation crawler built in Go. It replaces legacy Selenium scripts with native **Chrome DevTools Protocol (CDP)** browser automation, anti-bot stealth, and **direct Server-Sent Events (SSE) network stream interception**.

---

## Key Features

- 🚀 **Zero External Drivers**: No `chromedriver.exe` needed. Built on native CDP (`go-rod` + `stealth`).
- ⚡ **Real-Time Token Streaming**: Live typewriter effect in your terminal as tokens arrive.
- 🤖 **Headless by Default**: Fast, silent execution in the background without browser windows popping up.
- 🌐 **OpenAI-Compatible REST Server**: Exposes standard `POST /v1/chat/completions` so you can connect Cursor, LangChain, or your custom backends directly.
- 📜 **In-Conversation History**: Extract conversation turn history with `--history` or `/history`.
- 🔐 **Persistent Profiles or Pure Anonymous**: User profiles are preserved in `~/.chatgpt-spider/profile` so you log in once. Want unauthenticated public guest mode? Just pass `--anon`!
- 🆕 **New Chat Sessions**: Start a fresh thread with `--new-chat` or by typing `/new` inside interactive chat.
- 🔑 **Encrypted Session Token Storage**: Securely store your `__Secure-next-auth.session-token` (and chunked `.0`, `.1`) with AES-256 GCM using `chatgpt-spider auth set-token`.
- 💬 **Interactive Terminal REPL**: Chat naturally in your terminal with `chatgpt-spider chat`.
- ⚡ **Batch Processing**: Run hundreds of prompts from a text file with `chatgpt-spider batch -i prompts.txt -o results.json`.
- 🌐 **Direct Webhook Streaming**: Ingest conversation turns directly into your backend API via `--webhook-url`.
- 📝 **Multi-Format Export**: Export conversations to clean Markdown (`.md`) or formatted JSON (`.json`).

---

## Quick Start

### 1. Build from Source

```bash
cd d:\Projects\GolandProjects\spiders\chatgpt-spider
go build -o chatgpt-spider.exe .
```

### 2. Authentication & Guest Mode

You have complete flexibility:

#### Option A: Anonymous Guest Mode (No Login Required)
Want to use ChatGPT's free public tier without logging in or using your stored session? Just add `--anon`:
```bash
.\chatgpt-spider.exe prompt "What is the capital of France?" --anon
```

#### Option B: Store Session Token Securely (Logged In)
Copy your `__Secure-next-auth.session-token` (or chunked `.0` and `.1`) cookies and save them:
```bash
.\chatgpt-spider.exe auth set-token
```
Or pass directly on the command line:
```bash
.\chatgpt-spider.exe prompt "Hello!" -t "eyJhbGciOi..."
```

#### Option C: Log In Visually Once
Run with `--headless=false`:
```bash
.\chatgpt-spider.exe chat --headless=false
```
Chrome will open to ChatGPT. Complete your login once, and your session is saved in `~/.chatgpt-spider/profile`.

---

## Usage Examples

### 1. Live Terminal Chat (Typewriter Effect)
```powershell
.\chatgpt-spider.exe chat "hey whats up"
```
*(In chat mode, type `/history` to see previous turns, `/new` to start fresh, `/open <id>` to switch conversation threads, or `exit` to quit).*

### 2. Single Prompt (Headless by Default)
```powershell
.\chatgpt-spider.exe prompt "Explain goroutines in Go" -o response.md
```

### 3. Fetch In-Conversation History
To read the conversation history of a specific session thread without sending a new message:
```powershell
.\chatgpt-spider.exe prompt --chat-session-id "6a9edbf9-fffc-83e9-8216-4fb3e4e38f49" --history
```

### 4. Resume an Existing Conversation Thread
You can continue an existing conversation by passing its UUID or full URL:
```powershell
.\chatgpt-spider.exe prompt "Can you explain that last point further?" --chat-session-id "67c9b2e1-4567-89ab-cdef-0123456789ab"
```

### 5. OpenAI-Compatible REST API Server
Start the local server:
```powershell
.\chatgpt-spider.exe serve --port 8080
```
Then use it with standard `curl` or any OpenAI client:
```powershell
curl http://localhost:8080/v1/chat/completions `
  -H "Content-Type: application/json" `
  -d '{"messages": [{"role": "user", "content": "Explain binary search in 1 sentence"}]}'
```
Or with live SSE streaming:
```powershell
curl -N http://localhost:8080/v1/chat/completions `
  -H "Content-Type: application/json" `
  -d '{"stream": true, "messages": [{"role": "user", "content": "Write a short poem"}]}'
```

### 6. Batch Prompt File Execution
Given a `prompts.txt`:
```text
Write a haiku about computers
Explain photosynthesis in 2 sentences
What is Moore's law?
```

Execute them in sequence:
```powershell
.\chatgpt-spider.exe batch -i prompts.txt -o results.json -d 3
```

---

## Command Line Reference

```text
Flags:
      --anon                   Run in anonymous / guest mode without using saved credentials or profile
      --chat-session-id string Resume a specific existing conversation by ID or URL (e.g., 67c9b2e1-...)
      --headless               Run browser in headless mode (default true, set --headless=false to view browser)
      --history                Fetch and display the message history of the conversation
      --new-chat               Start a fresh conversation thread before sending prompt
      --debug                  Keep browser inspector / devtools open
  -t, --session-token string   ChatGPT __Secure-next-auth.session-token cookie
  -o, --output string          Output file path (.md or .json)
      --webhook-url string     Custom HTTP webhook endpoint to receive conversation turns
```

### Subcommands
- `prompt [text]`: Send a single prompt or view history with `--history`.
- `chat`: Open an interactive chat REPL in the terminal (`/history`, `/new`, `/open`).
- `serve`: Start an OpenAI-compatible REST server (`--port 8080`).
- `batch -i <file>`: Process a text file of prompts sequentially.
- `auth [set-token|status|clear]`: Manage encrypted session tokens.

---

## License
MIT License