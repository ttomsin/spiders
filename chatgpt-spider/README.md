# chatgpt-spider

**chatgpt-spider** is a high-performance native ChatGPT automation crawler built in Go. It replaces legacy Selenium scripts with native **Chrome DevTools Protocol (CDP)** browser automation, anti-bot stealth, and **direct Server-Sent Events (SSE) network stream interception**.

---

## Key Features

- 🚀 **Zero External Drivers**: No `chromedriver.exe` needed. Built on native CDP (`go-rod` + `stealth`).
- 📡 **Direct Network Stream Capture**: Intercepts `POST /backend-api/conversation` SSE JSON streams straight from the wire, extracting clean Markdown and text without relying on fragile React DOM classes.
- 🔐 **Persistent Browser Profile & Sessions**: User profiles are preserved in `~/.chatgpt-spider/profile` so you **only log in once**. Subsequent runs start pre-authenticated.
- 🔑 **Encrypted Session Token Storage**: Securely store your `__Secure-next-auth.session-token` with AES-256 GCM using `chatgpt-spider auth set-token`.
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

### 2. Authentication

You have two simple ways to authenticate:

#### Option A: Log In Visually Once (Recommended)
Simply run any command without `--headless`. Chrome will open to ChatGPT. Complete your login once, and your session is permanently saved in `~/.chatgpt-spider/profile`.

#### Option B: Store Session Token Securely
Copy your `__Secure-next-auth.session-token` cookie from your browser and save it:
```bash
.\chatgpt-spider.exe auth set-token
```
Or pass it directly on the command line:
```bash
.\chatgpt-spider.exe prompt "Hello!" -t "eyJhbGciOi..."
```

---

## Usage Examples

### Single Prompt Execution
```powershell
.\chatgpt-spider.exe prompt "Explain how goroutines and channels work in Go" -o response.md
```

### Interactive Terminal Chat
```powershell
.\chatgpt-spider.exe chat -o conversation.json
```

### Batch Prompt File Execution
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

### Stream Conversations to Your Backend API
```powershell
.\chatgpt-spider.exe prompt "Explain transformer attention mechanisms" --webhook-url "https://api.yourdomain.com/v1/chatgpt/ingest"
```

---

## Command Line Reference

```text
Flags:
  -t, --session-token string   ChatGPT __Secure-next-auth.session-token cookie
      --headless               Run browser in headless mode (default false)
      --debug                  Keep browser inspector open
  -o, --output string          Output file path (.md or .json)
      --webhook-url string     Custom HTTP webhook endpoint to receive conversation turns
```

### Subcommands
- `prompt <text>`: Send a single prompt and retrieve the response.
- `chat`: Open an interactive chat REPL in the terminal.
- `batch -i <file>`: Process a text file of prompts sequentially.
- `auth [set-token|status|clear]`: Manage encrypted session tokens.

---

## License
MIT License