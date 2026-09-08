# ChatGPT Spider - Roadmap & TODO List

This document tracks upcoming features and architectural enhancements for `chatgpt-spider`.

---

## 📋 Queued Features (Items 3 - 5)

### 1. 🌐 Web Search Toggle (`--search` / `--web`)
Enable ChatGPT's native search engine to retrieve live web data and citations.
- [ ] **CLI Flag**: Add `--search` (alias `--web`) to `prompt` command.
- [ ] **Interactive REPL**: Add `/search [on|off]` command in `chat` mode.
- [ ] **REST API**: Support `"web_search": true` or `"search": true` in `/v1/chat/completions` request body.
- [ ] **DOM Automation**:
  - Locate and click the web search toggle button in the composer toolbar (`button[aria-label*='Search']` / `button[data-testid='search-mode-button']`).
  - Verify toggle activation state before submitting prompt.
- [ ] **Citation & Source Extraction**:
  - Parse search query pills and citation hyperlinks returned in the assistant's response.
  - Export citation metadata in JSON outputs.

---

### 2. ⚡ Zero-Cold-Start Daemon Mode (Sub-Second Latency)
Eliminate browser startup overhead for instant prompt responses (<300ms).
- [ ] **Daemon Management CLI**:
  - `chatgpt-spider daemon start` (spins up background worker keeping Chrome warm at `https://chatgpt.com`).
  - `chatgpt-spider daemon stop`
  - `chatgpt-spider daemon status`
- [ ] **Local IPC / Socket Server**:
  - Lightweight local named pipe (Windows) / Unix socket (macOS/Linux) or localhost HTTP control plane.
- [ ] **CLI Transparent Fallback**:
  - When running `chatgpt-spider prompt "..."`, check if the daemon is running.
  - If daemon is active: dispatch prompt immediately via IPC to the warm tab and stream response.
  - If daemon is inactive: fallback gracefully to direct browser launch.
- [ ] **Connection & Tab Pooling**:
  - Auto-recycle tabs or trigger clean `NewChat` between isolated prompt requests.

---

### 3. 🔌 Third-Party Client Integration (Cursor, Open WebUI, LibreChat)
Certify `chatgpt-spider serve` as a seamless local OpenAI API provider.
- [ ] **Multi-Turn Message Context Aggregator**:
  - Parse OpenAI-style `messages: [...]` arrays containing full conversation histories (`system`, `user`, `assistant`).
  - Flatten previous turns into conversation context for fresh sessions or synchronize with existing conversation threads.
- [ ] **Models Endpoint (`/v1/models`) Expansion**:
  - Expose available models (`gpt-4o`, `gpt-4o-mini`, `o3-mini`, `o1`, `chatgpt-auto`).
- [ ] **Integration Guides & Presets**:
  - **Open WebUI**: Configuration guide and docker-compose snippet connecting to `http://host.docker.internal:8080/v1`.
  - **Cursor**: Instructions for setting OpenAI Base URL to `http://localhost:8080/v1`.
  - **LibreChat**: Custom provider configuration yaml.
  - **Continue.dev**: Config JSON snippet for VS Code / JetBrains.
- [ ] **Tool Use / Function Calling Passthrough**:
  - Explore simulation of JSON schema tool calls via format enforcement.

---

## 📌 Active Development Focus

- **Item 1**: 🧠 Dynamic Model Selection (`--model` / `/model`)
- **Item 2**: 🖼️ Real Multi-Modal File & Image Uploads (Vision / PDFs)
