# x-spider-ai

```text
       / _ \      __  __           ____        _     _                   _     ___ 
     \_\(_)/_/    \ \/ /          / ___| _ __ (_) __| | ___ _ __        / \   |_ _|
      _//o\\_      \  /  _____   \___ \| '_ \| |/ _` |/ _ \ '__|_____  / _ \   | | 
       /   \       /  \ |_____|   ___) | |_) | | (_| |  __/ |  |_____|/ ___ \  | | 
      /     \     /_/\_\         |____/| .__/|_|\__,_|\___|_|        /_/   \_\|___|
                                       |_|                                         
                [The Pluggable Autonomous Agent & AI Action Engine for X]
```

**x-spider-ai** is an agentic framework, Go SDK, and Model Context Protocol (MCP) server designed specifically for AI agents to interact with Twitter/X. It mimics authentic human behavior, providing LLMs and autonomous agents with the "hands and eyes" to read, post, reply, scroll, quote, bookmark, message, and engage on X.

> 📖 **Looking for full documentation and command tutorials? Check out [`HOW-TO-USE-X-SPIDER-AI.md`](HOW-TO-USE-X-SPIDER-AI.md).**

---

## Architecture & Capabilities

```
+-------------------------------------------------------------+
|                      AI Agent Layers                        |
|   (Claude Desktop / Cursor / LangChain / Gemini / AutoGen)  |
+-------------------------------------------------------------+
        |                                       |
    [MCP Protocol (stdio)]              [REST / JSON HTTP]
        |                                       |
+-------------------------------------------------------------+
|                      x-spider-ai Core                       |
|   - Go SDK Package (pkg/xspiderai)                          |
|   - Native CDP Browser Engine (Go-Rod + Stealth)            |
|   - Request Hijacker & GraphQL Response Interceptor         |
|   - Encrypted Session Store (AES-256 GCM)                   |
+-------------------------------------------------------------+
        |                                       |
        v                                       v
[DOM Interactions / Emulation]         [Live GraphQL Streams]
(Typing, clicks, uploads, scrolls)     (Interception & Token harvesting)
```

### Key Highlights
- 🤖 **100% Pluggable for AI**: Built directly on the **Model Context Protocol (MCP)** and exposed via REST / JSON endpoints or direct Go SDK imports.
- 📜 **Human-like Scroll & Thread Traversal**: Real-time scrolling through long threads, fetching all comments/replies dynamically without relying on fragile pagination tokens.
- 🖼️ **Media Extraction**: Extracts full-resolution image URLs, highest-bitrate MP4 video streams, and GIFs from tweets.
- ⚡ **Network Hijacking**: Intercepts underlying Twitter GraphQL endpoints (`TweetDetail`, `SearchTimeline`, `UserTweets`) and harvests Bearer/CSRF tokens on the fly.
- 💬 **Interactive Human Actions**:
  - `x_post_tweet`: Compose tweets, threads, and media attachments.
  - `x_read_thread`: Read main tweet + entire comment tree.
  - `x_quote_tweet`: Quote tweet any existing status.
  - `x_like_tweet` / `x_unlike_tweet`: Like and unlike.
  - `x_retweet` / `x_unretweet`: Repost and undo repost.
  - `x_bookmark_tweet` / `x_unbookmark_tweet`: Manage bookmarks.
  - `x_delete_tweet`: Delete authentic user tweets.
  - `x_follow_user` / `x_unfollow_user`: Manage follower graph.
  - `x_send_direct_message`: Send direct messages (DMs).
  - `x_scroll_page`: Emulate human scrolling to trigger network interception.

---

## Quick Start

### 1. Download Standalone Binary (No Go Required!)
Download the precompiled binary for your operating system directly from [**Releases**](https://github.com/ttomsin/spiders/releases):

| OS / Architecture | Binary Name | Run Command |
| :--- | :--- | :--- |
| **Windows (64-bit)** | `xsai-windows-amd64.exe` | `.\xsai.exe` |
| **Windows (ARM64)** | `xsai-windows-arm64.exe` | `.\xsai.exe` |
| **Linux (x86_64)** | `xsai-linux-amd64` | `./xsai` |
| **Linux (ARM64)** | `xsai-linux-arm64` | `./xsai` |
| **macOS (Apple Silicon)** | `xsai-darwin-arm64` | `./xsai` |
| **macOS (Intel)** | `xsai-darwin-amd64` | `./xsai` |

Rename the downloaded binary to `xsai` (or `xsai.exe` on Windows) and place it in your `PATH` (such as `/usr/local/bin` on Linux/macOS).

---

### 2. Build Locally (Alternative)
If you have Go 1.24+ installed:
```bash
# Windows
.\scripts\build-all.ps1

# Linux / macOS
chmod +x ./scripts/build-all.sh
./scripts/build-all.sh
```

---

### 3. Initialize the Secure Vault
Initialize your encrypted SQLite database and machine-derived AES-256 keys:
```bash
xsai start
```

---

### Windows Defender & Smart App Control Note
When downloading or locally compiling a brand-new executable on Windows 11, **Smart App Control (SAC)** or **Windows Defender** may block untrusted unsigned binaries with:
```text
Program 'xsai.exe' failed to run: An Application Control policy has blocked this file
```

**How to unblock on Windows:**
1. **Unblock the file via PowerShell**:
   ```powershell
   Unblock-File .\xsai.exe
   ```
2. **Or run via Go directly during development**:
   ```powershell
   go run .\cmd\x-spider-ai start
   ```
3. **Or add a directory exclusion** in Windows Security &rarr; *Virus & threat protection* &rarr; *Manage settings* &rarr; *Exclusions*.

---

## Authentication & Setup

### Why Using `auth_token` is Essential (Anti-Bot & Stealth)

Twitter/X implements aggressive fingerprinting and anti-bot defenses (Cloudflare, Arkose Labs, Kasada, and Cross-Origin-Opener-Policy protection) directly on its authentication endpoints (`x.com/login` and SSO gateways). 

When automated browsers (Selenium, Puppeteer, Playwright, CDP) attempt to navigate through the interactive login UI:
1. **SSO & Pop-up Blocking**: Third-party login popups (Google/Apple SSO) are often blocked or disrupted by strict security policies.
2. **Bot-Challenges & Account Flagging**: The login form triggers silent behavioral telemetry that detects browser automation, resulting in errors like *"An unexpected error occurred. Please try again"* or triggering SMS/email security locks.
3. **Session Instability**: UI credentials can require recurrent CAPTCHAs, 2FA prompts, or device confirmations.

#### The Cookie Injection Advantage
By extracting your **`auth_token`** cookie from an already authenticated browser session:
- **Zero Bot Triggers**: You bypass Twitter's login wall and anti-bot evaluation entirely.
- **Natural Authenticated Context**: Twitter immediately recognizes the session as a legitimate, logged-in browser navigating `x.com/home`.
- **Permanent Stealth**: The session is saved once, encrypted locally, and reused silently across headless AI runs without requiring repeated logins.

---

### Step-by-Step: Extracting Your `auth_token`

1. Open your regular browser (Chrome, Brave, Edge, or Firefox) where you are already signed in to [x.com](https://x.com).
2. Press **`F12`** (or right-click anywhere and select **Inspect**) to open Developer Tools.
3. Navigate to:
   - **Chrome / Brave / Edge**: Go to the **Application** tab > Expand **Cookies** in the left sidebar > Click `https://x.com`.
   - **Firefox**: Go to the **Storage** tab > Expand **Cookies** > Click `https://x.com`.
4. Locate the cookie named **`auth_token`** and copy its **Value** (a 40-character hex string).
5. *(Optional)* Locate the cookie named **`ct0`** and copy its Value (CSRF token).

---

### Saving Your Session to `x-spider-ai`

Run the login command in PowerShell or terminal:

```bash
cd x-spider-ai
.\x-spider-ai.exe login "<YOUR_AUTH_TOKEN>"
```

Or pass both `auth_token` and `ct0`:
```bash
.\x-spider-ai.exe login "<YOUR_AUTH_TOKEN>" "<YOUR_CT0>"
```

### SQLite Encrypted Storage

All session credentials (`auth_token`, `ct0`, bearer tokens, and session cookies) are automatically encrypted with machine-derived **AES-256 GCM** encryption and stored in a local SQLite database:

```text
~/.x-spider-ai/sessions.db
```

The database is created automatically on your first login. You can inspect or manage it anytime:

- **Check session and database status**:
  ```bash
  .\x-spider-ai.exe db info
  ```
- **Log out (clears credentials from SQLite)**:
  ```bash
  .\x-spider-ai.exe logout
  ```
- **Permanently delete the SQLite database file**:
  ```bash
  .\x-spider-ai.exe db delete
  ```

### Multi-Account Vault & Switching
You can register and store multiple Twitter/X accounts in the encrypted SQLite vault and switch between them dynamically or target specific accounts per-action:

- **Add an account by ID**:
  ```bash
  .\x-spider-ai.exe login <auth_token> <ct0> --id bot_alpha --handle @_alpha_bot
  ```
- **List all saved accounts**:
  ```bash
  .\x-spider-ai.exe accounts list
  ```
- **Switch the active default account**:
  ```bash
  .\x-spider-ai.exe accounts switch bot_alpha
  ```
- **Target a specific account on any command**:
  ```bash
  .\x-spider-ai.exe post "Hello from bot alpha!" --account bot_alpha
  .\x-spider-ai.exe like 2107167470134133132 --account main
  ```

---

## Hands-On Testing Guide (Try These Now!)

You can test every capability right away using this live community tweet:
- **Target Tweet**: [`https://x.com/i/status/2107167470134133132`](https://x.com/i/status/2107167470134133132)
- **Target Tweet ID**: `2107167470134133132`
- **Target Handle**: `@_ttomsin`

### 1. Like the Tweet
```bash
.\x-spider-ai.exe like 2107167470134133132 --headless=false
```

### 2. Retweet / Repost
```bash
.\x-spider-ai.exe retweet 2107167470134133132
```

### 3. Quote Tweet with Commentary
```bash
.\x-spider-ai.exe quote 2107167470134133132 "Testing out x-spider-ai automated agent! 🕷️🤖" --headless=false
```

### 4. Bookmark the Tweet
```bash
.\x-spider-ai.exe bookmark 2107167470134133132
```

### 5. Read Thread & Comments via HTTP Server
Start the server:
```bash
.\x-spider-ai.exe server --port 8080
```
Then fetch the full discussion thread and all replies:
```bash
curl "http://localhost:8080/api/v1/tweets/thread?tweet_id=2107167470134133132"
```

### 6. Follow the Creator
Via HTTP API:
```bash
curl -X POST http://localhost:8080/api/v1/users/follow -H "Content-Type: application/json" -d "{\"screen_name\": \"_ttomsin\"}"
```

---

## Usage Modes

### MCP Server (For Claude, Cursor, Open WebUI, Antigravity)
Add `x-spider-ai` to your MCP configuration (e.g., `claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "x-spider-ai": {
      "command": "d:\\Projects\\GolandProjects\\spiders\\x-spider-ai\\x-spider-ai.exe",
      "args": ["mcp", "--headless=true"]
    }
  }
}
```

Now any LLM can call:
- `x_post_tweet(text="Hello world from AI!")`
- `x_read_thread(tweet_id="2107167470134133132", max_scrolls=5)`
- `x_like_tweet(tweet_id="2107167470134133132")`
- `x_retweet(tweet_id="2107167470134133132")`
- `x_send_direct_message(screen_name="_ttomsin", text="Hi from x-spider-ai!")`
- `x_search_tweets(query="golang ai", tab="Latest")`
- `x_discover(queries=["who started saying no wahala", "where did no wahala come from"], since="2020-01-01", until="2024-01-01", min_engagement=10, sort="engagement", include_replies=true)`
- `x_list_accounts()`
- `x_switch_account(account_id="bot_alpha")`

### REST / JSON API (For Python, Node.js, LangChain, AutoGen)
Launch the standalone server:
```bash
.\x-spider-ai.exe server --port 8080
```

Available REST Endpoints:
- `POST /api/v1/auth/login`
- `GET  /api/v1/accounts`
- `POST /api/v1/accounts/switch`
- `DELETE /api/v1/accounts?account_id=...`
- `POST /api/v1/tweets/post`
- `POST /api/v1/tweets/like`
- `POST /api/v1/tweets/unlike`
- `POST /api/v1/tweets/retweet`
- `POST /api/v1/tweets/bookmark`
- `POST /api/v1/tweets/delete`
- `GET  /api/v1/tweets/thread?tweet_id=...`
- `GET  /api/v1/tweets/search?query=...&tab=...`
- `POST /api/v1/tweets/discover`
- `GET  /api/v1/users/timeline?screen_name=...`
- `POST /api/v1/users/follow`
- `POST /api/v1/messages/send`
- `POST /api/v1/browser/scroll`

### Go SDK Package
Import `x-spider-ai/pkg/xspiderai` directly into any Go application:

```go
package main

import (
	"fmt"
	"log"
	"x-spider-ai/pkg/xspiderai"
)

func main() {
	client, err := xspiderai.NewClient(xspiderai.ClientOptions{
		Headless: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Post tweet
	res, err := client.PostTweet(xspiderai.PostTweetOptions{
		Text: "Building autonomous agents with Go and x-spider-ai! 🚀",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Action result: %+v\n", res)

	// Read thread with replies and media
	thread, err := client.ReadThread("2107167470134133132", xspiderai.ScrollOptions{MaxScrolls: 4})
	if err == nil {
		fmt.Printf("Main Tweet: %s\n", thread.MainTweet.FullText)
		for _, reply := range thread.Replies {
			fmt.Printf("Reply from @%s: %s\n", reply.Author.ScreenName, reply.FullText)
		}
	}
}
```
