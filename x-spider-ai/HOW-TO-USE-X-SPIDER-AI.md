# HOW TO USE X-SPIDER-AI

```text
       / _ \      __  __           ____        _     _                   _     ___ 
     \_\(_)/_/    \ \/ /          / ___| _ __ (_) __| | ___ _ __        / \   |_ _|
      _//o\\_      \  /  _____   \___ \| '_ \| |/ _` |/ _ \ '__|_____  / _ \   | | 
       /   \       /  \ |_____|   ___) | |_) | | (_| |  __/ |  |_____|/ ___ \  | | 
      /     \     /_/\_\         |____/| .__/|_|\__,_|\___|_|        /_/   \_\|___|
                                       |_|                                         
                [The Pluggable Autonomous Agent & AI Action Engine for X]
```

Welcome to **x-spider-ai**! This comprehensive manual guides you through every single feature and shows how to use it across all 3 interfaces:
1. **Command Line (CLI)**
2. **Model Context Protocol (MCP)** (for Claude, Cursor, Open WebUI, and custom LLM agents)
3. **HTTP REST / JSON API** (for Python, LangChain, AutoGen, CrewAI, Node.js)
4. **Native Go SDK** (`pkg/xspiderai`)

---

## Table of Contents
- [Authentication & Setup](#authentication--setup)
- [Feature Matrix Across Interfaces](#feature-matrix-across-interfaces)
- [CLI Reference](#cli-reference)
- [Model Context Protocol (MCP) Tools](#model-context-protocol-mcp-tools)
- [REST / JSON API Reference](#rest--json-api-reference)
- [Native Go SDK Reference](#native-go-sdk-reference)
- [Session & Database Management](#session--database-management)

---

## Authentication & Setup

Twitter/X aggressively checks automated logins on `x.com/login`. To maintain absolute stealth and bypass bot detection, `x-spider-ai` uses **AES-256 GCM encrypted SQLite session storage** with your `auth_token`.

### Step 1: Extract Your Token
1. Open your regular browser (Chrome, Brave, Edge, Firefox) where you are logged into [x.com](https://x.com).
2. Press **`F12`** (Developer Tools).
3. Go to **Application** (or **Storage** in Firefox) > **Cookies** > `https://x.com`.
4. Copy the value of **`auth_token`** (a 40-character hex string).
5. *(Optional)* Copy the value of **`ct0`** (CSRF token).

### Step 2: Save Your Session
Run this once in PowerShell or your terminal:

```powershell
cd d:\Projects\GolandProjects\spiders\x-spider-ai
.\x-spider-ai.exe login "<YOUR_AUTH_TOKEN>"
```

Or with `ct0`:
```powershell
.\x-spider-ai.exe login "<YOUR_AUTH_TOKEN>" "<YOUR_CT0>"
```

Your session is encrypted using machine-specific keys and saved into:
`~/.x-spider-ai/sessions.db`. You are now ready to run everything headlessly!

---

## Feature Matrix Across Interfaces

| Feature | CLI | MCP Tool | REST API Endpoint | Go SDK Method |
| :--- | :--- | :--- | :--- | :--- |
| **Post Tweet** | `post <text>` | `x_post_tweet` | `POST /api/v1/tweets/post` | `client.PostTweet(...)` |
| **Post with Media** | `post <text> -m <path>` | `x_post_tweet(media_file_paths)` | `POST /api/v1/tweets/post` | `client.PostTweet(media_file_paths)` |
| **Quote Tweet** | `quote <id> [text]` | `x_post_tweet(quote_tweet_id)` | `POST /api/v1/tweets/post` | `client.QuoteTweet(id, text)` |
| **Reply to Tweet** | *(via quote/reply)* | `x_post_tweet(in_reply_to_id)` | `POST /api/v1/tweets/post` | `client.ReplyTweet(id, text)` |
| **Like Tweet** | `like <id>` | `x_like_tweet` | `POST /api/v1/tweets/like` | `client.LikeTweet(id)` |
| **Unlike Tweet** | `unlike <id>` | `x_unlike_tweet` | `POST /api/v1/tweets/unlike` | `client.UnlikeTweet(id)` |
| **Retweet / Repost** | `retweet <id>` | `x_retweet` | `POST /api/v1/tweets/retweet` | `client.Retweet(id)` |
| **Bookmark Tweet** | `bookmark <id>` | `x_bookmark_tweet` | `POST /api/v1/tweets/bookmark` | `client.BookmarkTweet(id)` |
| **Follow User** | `follow <handle>` | `x_follow_user` | `POST /api/v1/users/follow` | `client.FollowUser(handle)` |
| **Unfollow User** | `unfollow <handle>` | `x_unfollow_user` | `POST /api/v1/users/unfollow` | `client.UnfollowUser(handle)` |
| **Check Any Profile**| `profile <handle>` | `x_get_profile` | `GET /api/v1/users/profile` | `client.GetProfile(handle)` |
| **Check My Profile** | `me` | `x_get_my_profile` | `GET /api/v1/users/me` | `client.GetMyProfile()` |
| **Read Thread & Replies** | *(API / MCP / SDK)* | `x_read_thread` | `GET /api/v1/tweets/thread` | `client.ReadThread(id, opts)` |
| **Search Tweets** | *(API / MCP / SDK)* | `x_search_tweets` | `GET /api/v1/tweets/search` | `client.SearchTweets(q, tab, opts)` |
| **Send Direct Message**| *(API / MCP / SDK)* | `x_send_direct_message` | `POST /api/v1/messages/send` | `client.SendDirectMessage(to, txt)` |
| **Scroll Current Page**| *(API / MCP / SDK)* | `x_scroll_page` | `POST /api/v1/browser/scroll` | `client.Scroll(opts)` |

---

## CLI Reference

All CLI commands run in background **headless mode** by default. To watch the browser perform the action visually, append `--headless=false`.

### 1. Check Logged-in Profile
```powershell
.\x-spider-ai.exe me
```
*Output:*
```text
Logged in as @_ttomsin:
  Name: Thompson
  Bio: Building autonomous AI systems.
  Followers: 120
  Following: 85
  Posts: 42
```

### 2. Check Another User's Profile
```powershell
.\x-spider-ai.exe profile _ttomsin
```

### 3. Post a Tweet
```powershell
.\x-spider-ai.exe post "Building autonomous agents on Twitter with x-spider-ai! 🕷️🚀"
```
*Output:*
```text
Posted successfully! Tweet ID: 2107167470134133132 | URL: https://x.com/i/status/2107167470134133132
```

### 4. Post a Tweet with Media (Images or Video)
```powershell
.\x-spider-ai.exe post "Here is the new demo architecture!" --media "C:\Users\User\Pictures\architecture.png"
```
*(You can pass multiple files: `-m "img1.png" -m "img2.jpg"`)*

### 5. Quote a Tweet (with or without comments)
```powershell
.\x-spider-ai.exe quote 2107167470134133132 "Strongly agree with this perspective! 🔥"
```
Quote without text:
```powershell
.\x-spider-ai.exe quote 2107167470134133132
```

### 6. Like & Unlike Tweets
```powershell
.\x-spider-ai.exe like 2107167470134133132
.\x-spider-ai.exe unlike 2107167470134133132
```

### 7. Retweet / Repost
```powershell
.\x-spider-ai.exe retweet 2107167470134133132
```

### 8. Bookmark
```powershell
.\x-spider-ai.exe bookmark 2107167470134133132
```

### 9. Follow & Unfollow Users
```powershell
.\x-spider-ai.exe follow _ttomsin
.\x-spider-ai.exe unfollow _ttomsin
```

---

## Model Context Protocol (MCP) Tools

To use `x-spider-ai` inside **Claude Desktop**, **Cursor**, or custom AI agents, register it in your MCP configuration file:

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

### Available MCP Tools for the AI
- `x_post_tweet(text, [in_reply_to_id], [quote_tweet_id], [media_file_paths])`
- `x_like_tweet(tweet_id)`
- `x_unlike_tweet(tweet_id)`
- `x_retweet(tweet_id)`
- `x_bookmark_tweet(tweet_id)`
- `x_delete_tweet(tweet_id)`
- `x_follow_user(screen_name)`
- `x_unfollow_user(screen_name)`
- `x_get_profile(screen_name)`
- `x_get_my_profile()`
- `x_read_thread(tweet_id, [max_scrolls])`
- `x_search_tweets(query, [tab], [max_scrolls])`
- `x_read_user_timeline(screen_name, [max_scrolls])`
- `x_send_direct_message(screen_name, text)`
- `x_scroll_page([scroll_count], [delay_ms])`

---

## REST / JSON API Reference

Start the local server:
```powershell
.\x-spider-ai.exe server --port 8080
```

### Endpoints

#### 1. Check Server Health
`GET http://localhost:8080/health`

#### 2. Get Authenticated User Profile
`GET http://localhost:8080/api/v1/users/me`

#### 3. Get User Profile by Handle
`GET http://localhost:8080/api/v1/users/profile?screen_name=_ttomsin`

#### 4. Read Tweet Thread & Replies with Scrolling
`GET http://localhost:8080/api/v1/tweets/thread?tweet_id=2107167470134133132`

#### 5. Search Tweets
`GET http://localhost:8080/api/v1/tweets/search?query=golang+ai&tab=Latest`

#### 6. Read User Timeline
`GET http://localhost:8080/api/v1/users/timeline?screen_name=_ttomsin`

#### 7. Post a Tweet (or Quote / Reply)
`POST http://localhost:8080/api/v1/tweets/post`
```json
{
  "text": "Hello world from API!",
  "quote_tweet_id": "2107167470134133132",
  "media_file_paths": ["C:\\path\\to\\image.png"]
}
```

#### 8. Like a Tweet
`POST http://localhost:8080/api/v1/tweets/like`
```json
{
  "tweet_id": "2107167470134133132"
}
```

#### 9. Follow a User
`POST http://localhost:8080/api/v1/users/follow`
```json
{
  "screen_name": "_ttomsin"
}
```

#### 10. Send a Direct Message (DM)
`POST http://localhost:8080/api/v1/messages/send`
```json
{
  "screen_name": "_ttomsin",
  "text": "Hey, love your autonomous agent project!"
}
```

---

## Native Go SDK Reference

Import the package directly in your Go code:

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

	// 1. Check profile
	me, err := client.GetMyProfile()
	if err == nil {
		fmt.Printf("Logged in as @%s\n", me.ScreenName)
	}

	// 2. Post a tweet with media
	res, err := client.PostTweet(xspiderai.PostTweetOptions{
		Text: "Automated tweet with Go SDK!",
		MediaFilePaths: []string{"C:\\path\\to\\image.png"},
	})
	if err == nil {
		fmt.Printf("Tweet ID: %s | URL: %s\n", res.TweetID, res.TweetURL)
	}

	// 3. Read thread and all comment replies
	thread, err := client.ReadThread("2107167470134133132", xspiderai.ScrollOptions{MaxScrolls: 4})
	if err == nil {
		fmt.Println("Main:", thread.MainTweet.FullText)
		for _, reply := range thread.Replies {
			fmt.Printf("@%s: %s\n", reply.Author.ScreenName, reply.FullText)
		}
	}
}
```

---

## Session & Database Management

- **View active SQLite database and session details**:
  ```powershell
  .\x-spider-ai.exe db info
  ```
- **Log out (clears credentials from SQLite)**:
  ```powershell
  .\x-spider-ai.exe logout
  ```
- **Completely delete the local SQLite database file**:
  ```powershell
  .\x-spider-ai.exe db delete
  ```
