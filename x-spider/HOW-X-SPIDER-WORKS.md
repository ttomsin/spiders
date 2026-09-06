# How x-spider Works: Architecture and Code Guide

Welcome to the internal engineering guide for **x-spider**. This document provides an in-depth, step-by-step explanation of how the crawler is built, how data flows through the system, and how each component operates under the hood.

---

## 1. High-Level Architecture & Core Philosophy

Traditional web scrapers scrape the DOM (HTML tree) by querying elements using CSS selectors or XPath expressions like `div.tweet-text` or `span.username`. This approach is notoriously fragile on Twitter/X because:
1. CSS class names are generated dynamically by React/styled-components (e.g., `css-175oi2r`, `r-18u37iz`).
2. HTML element trees frequently change across app updates.
3. DOM extraction consumes significant CPU and memory when thousands of elements are loaded.

### The x-spider Approach: CDP Network Interception
Instead of inspecting HTML elements, **x-spider** uses Chrome DevTools Protocol (CDP) through [`go-rod`](https://github.com/go-rod/rod) to **intercept Twitter's internal GraphQL API responses directly from the network stream**.

```
   ┌────────────────────────────────────────────────────────┐
   │                       x-spider                         │
   │  ┌──────────────┐     ┌──────────────┐                 │
   │  │ Configuration│     │ Secure Auth  │                 │
   │  │(YAML/CLI/Env)│     │  (AES-GCM)   │                 │
   │  └──────┬───────┘     └──────┬───────┘                 │
   │         └──────────┬─────────┘                         │
   │                    ▼                                   │
   │           ┌─────────────────┐                          │
   │           │  Crawler Engine │                          │
   │           └────────┬────────┘                          │
   └────────────────────┼───────────────────────────────────┘
                        ▼
       ┌───────────────────────────────────┐
       │   Chromium (Stealth Automated)    │
       │                                   │
       │   1. Injects `auth_token` Cookie  │
       │   2. Navigates to x.com           │
       │   3. Types Search & Scrolls       │
       └────────────────┬──────────────────┘
                        │
                        ▼
       ┌───────────────────────────────────┐
       │     Network Hijack Router         │
       ├───────────────────────────────────┤
       │ ⛔ Block Media: .jpg, .mp4, etc.  │ ──► Saves 80% Bandwidth & RAM
       │ 📦 Intercept: SearchTimeline /    │
       │               TweetDetail         │ ──► Raw GraphQL JSON
       └────────────────┬──────────────────┘
                        │
                        ▼
       ┌───────────────────────────────────┐
       │      JSON Extraction Engine       │
       ├───────────────────────────────────┤
       │ • Extract tweet ID, user, text    │
       │ • Format date to ISO 8601 UTC     │
       │ • Deduplicate tweet IDs           │
       └────────────────┬──────────────────┘
                        │
                        ▼
       ┌───────────────────────────────────┐
       │          Export Pipeline          │
       ├───────────────────────────────────┤
       │ • CSV (RFC-4180, sorted headers)  │
       │ • Excel (.xlsx via excelize)      │
       │ • Gephi Graph (source, target)    │
       └───────────────────────────────────┘
```

---

## 2. Directory Structure and Responsibilities

```
x-spider/
├── cmd/                       # CLI commands and orchestration (Cobra)
│   ├── root.go                # Banner, CLI flag definitions, execution pipeline
│   ├── crawl.go               # 'crawl' subcommand
│   ├── auth.go                # 'auth' subcommands (set-token, status, clear)
│   ├── gephi.go               # 'gephi' network edge list converter
│   └── gephi_test.go          # Unit tests for Gephi converter
│
├── internal/
│   ├── auth/                  # Secure local credential storage & token pool
│   │   ├── auth.go            # AES-256 GCM encryption using machine-derived key
│   │   ├── auth_test.go       # Encryption and token masking tests
│   │   ├── pool.go            # Multi-account token pool & live rotation
│   │   └── pool_test.go       # Token pool tests
│   │
│   ├── checkpoint/            # Fault-tolerant crawl recovery
│   │   ├── checkpoint.go      # Saves/loads state from .xspider-checkpoint.json
│   │   └── checkpoint_test.go # Checkpoint tests
│   │
│   ├── chunker/               # Automated timeframe slicing
│   │   ├── chunker.go         # Slices date ranges (monthly, weekly, daily)
│   │   └── chunker_test.go    # Chunker date math tests
│   │
│   ├── cleaner/               # NLP text pre-cleaning pipeline
│   │   ├── cleaner.go         # Strips URLs, @mentions, emojis, filters min length
│   │   └── cleaner_test.go    # Text cleaning unit tests
│   │
│   ├── config/                # Configuration management
│   │   └── config.go          # YAML parsing, flag merging, environment overrides
│   │
│   ├── crawler/               # Core browser & network scraping logic
│   │   ├── crawler.go         # Browser lifecycle, cookie injection, crawl loop
│   │   ├── network.go         # Hijack router, media blocking, JSON listener
│   │   └── page.go            # Scrolling routines, DOM cleaner, keyword typing
│   │
│   ├── exporter/              # Output file generators
│   │   ├── csv.go             # RFC-4180 CSV writer with sorted alphabetical headers
│   │   ├── excel.go           # Excel (.xlsx) generator using excelize/v2
│   │   ├── json.go            # Formatted JSON array generator
│   │   ├── jsonl.go           # JSON Lines streaming generator (for LLM training)
│   │   ├── sqlite.go          # Pure-Go SQLite database exporter (zero CGO)
│   │   └── exporter_test.go   # Exporter unit tests
│   │
│   ├── model/                 # Data contracts & JSON unmarshalling
│   │   ├── tweet.go           # Twitter GraphQL timeline structures & parsing
│   │   └── tweet_test.go      # JSON unmarshalling and date conversion tests
│   │
│   ├── notifier/              # Webhook alerting system
│   │   ├── notifier.go        # Discord, Telegram, and generic HTTP POST webhooks
│   │   └── notifier_test.go   # Webhook dispatch unit tests
│   │
│   ├── prompt/                # Terminal interactive prompts
│   │   └── prompt.go          # Interactive password mask, number input, format picker
│   │
│   └── proxy/                 # Rotating proxy pool manager
│       ├── pool.go            # Thread-safe round-robin proxy rotation
│       └── pool_test.go       # Proxy pool unit tests
│
├── config.example.yaml        # Template configuration file
├── go.mod                     # Go 1.26 module definition
├── main.go                    # Entrypoint binary wrapper
└── README.md                  # User manual
```

---

## 3. Step-by-Step Lifecycle of a Crawl

### Phase 1: Configuration Resolution & Precedence
When you run `x-spider`, it aggregates configuration settings using a strict hierarchy of precedence:
1. **CLI Flags**: Passed explicitly via flags (e.g. `-s "golang" -l 50`).
2. **Environment Variables**: Such as `DEV_ACCESS_TOKEN`, `HEADLESS_MODE`.
3. **YAML File**: Loaded from `--config <path>` or local `./config.yaml`.
4. **Persistent Credentials**: Stored securely on your machine via `x-spider auth set-token`.
5. **Interactive Prompts**: If any required field (`auth_token`, `search_keyword`) is still missing, the CLI asks interactively.
6. **Defaults**: Predefined safe defaults (`limit: 10`, `delay: 3s`, `tab: TOP`, `format: csv`).

### Phase 2: Secure Persistent Authentication (`internal/auth`)
To avoid having to repeatedly type or hardcode your Twitter `auth_token`:
- `x-spider auth set-token` stores the token inside your user home directory: `~/.x-spider/credentials.json`.
- The token is **never stored in plaintext**.
- It is encrypted with **AES-256 GCM** using a key dynamically derived from machine attributes (hostname, current operating system user, and an internal salt).
- File permissions are restricted to `0600` (readable only by your user account).

### Phase 3: Browser Setup & Anti-Bot Stealth (`internal/crawler`)
- Chromium is launched via `go-rod` launcher.
- The `go-rod/stealth` wrapper overrides `navigator.webdriver`, masks Chrome DevTools Protocol fingerprints, and emulates realistic browser characteristics to avoid bot flags.
- Viewport is set to `1240 x 1080`.
- The `auth_token` cookie is injected with strict attributes:
  ```go
  Domain: ".x.com"
  Path: "/"
  Secure: true
  HTTPOnly: true
  SameSite: Strict
  ```

### Phase 4: Network Interception & Media Blocker (`internal/crawler/network.go`)
Before visiting any URL, a network request router (`HijackRequests`) is attached to the browser page:
1. **Media Blocking**:
   Every outgoing request URL is checked against `BlockedExtensions` (`.jpg`, `.png`, `.mp4`, `.gif`, `format=jpg`, etc.).
   If matched, the request is aborted immediately (`proto.NetworkErrorReasonBlockedByClient`).
   *Benefit*: Prevents downloading heavy images/videos during fast scrolling, saving ~80% network bandwidth and keeping memory consumption low.
2. **API Payload Interception**:
   Any request containing `SearchTimeline` or `TweetDetail` in its URL is loaded via `ctx.LoadResponse(http.DefaultClient, true)`.
   Its raw body is captured and routed to a Go channel (`dataChan`) for parsing.

### Phase 5: Query Synthesis (`internal/crawler/page.go`)
When searching keywords with dates:
- If `--from 01-01-2026` is supplied, it formats to `since:2026-01-01`.
- If `--to 01-02-2026` is supplied, it formats to `until:2026-02-01`.
- It navigates to `https://x.com/search-advanced` (or `https://x.com/search-advanced?f=live` for the `LATEST` tab).
- It locates `input[name="allOfTheseWords"]`, fills the query, and presses `Enter`.

### Phase 6: The Crawl & Scroll Loop (`internal/crawler/crawler.go`)
The crawler enters an event loop:
1. **Waiting for responses**:
   It listens on `interceptor.DataChannel()`.
   When a GraphQL response arrives, it parses the JSON:
   - `search_by_raw_query.search_timeline.timeline.instructions[...].entries` (Search mode)
   - `threaded_conversation_with_injections_v2.instructions[...].entries` (Detail mode)
2. **Data Extraction & Cleaning**:
   - Promoted ads (`entryId` contains "promoted") are discarded.
   - Author username and user location are resolved.
   - Twitter's timestamp format (`Mon Jan 02 15:04:05 -0700 2006`) is parsed and standardized to ISO 8601 UTC (`2026-01-02T15:04:05.000Z`).
   - In detail mode, leading `@mentions` are trimmed from replies for cleaner NLP analysis.
   - Deduplication: `seenTweetIDs[id_str]` ensures no duplicate tweets are recorded.
3. **Saving to Disk**:
   - Appends rows immediately to the selected exporter (CSV or XLSX) so you never lose progress if interrupted.
4. **Scrolling**:
   - If no response arrives after 1.5 seconds, it executes `ScrollDown(page)`.
   - In `ScrollDown`, it triggers `window.scrollTo({ top: document.body.scrollHeight, behavior: 'smooth' })` and actively purges `<div data-testid="tweetPhoto">` elements from the DOM to reclaim browser memory.
5. **Progressive Delays**:
   - Every 20 tweets: sleeps `delay_seconds` (default 3s).
   - Every 100 tweets: sleeps `delay_every_100_tweets` (default 10s) to simulate natural reading pauses.

### Phase 7: Rate Limit Handling & Tab Fallback
- **Rate Limits**:
  If Twitter responds with a `rate limit` error, `CalculateForRateLimit` computes a backoff duration:
  `timeout = (2 * attempt * 60s) + 60s` (up to 10 minutes).
  The crawler pauses, then clicks the Twitter UI "Retry" button.
- **Tab Fallback**:
  If 0 tweets were found on the `TOP` tab and the page shows "No results for", the crawler automatically switches to the `LATEST` tab (`https://x.com/search-advanced?f=live`) to ensure recent tweets aren't missed.

---

## 4. Output Schemas & Data Formatting

Both CSV and Excel exporters write the same 15 columns sorted alphabetically:

| Field Name | Type | Description |
|------------|------|-------------|
| `conversation_id_str` | string | Thread identifier |
| `created_at` | string | ISO 8601 UTC timestamp |
| `favorite_count` | int | Total like count |
| `full_text` | string | Complete tweet text |
| `id_str` | string | Unique Tweet ID |
| `image_url` | string | First media image URL (if attached) |
| `in_reply_to_screen_name` | string | Target username if tweet is a reply |
| `lang` | string | Language code (e.g. `en`, `id`, `ja`) |
| `location` | string | Author location from their profile |
| `quote_count` | int | Total quote count |
| `reply_count` | int | Total reply count |
| `retweet_count` | int | Total retweet count |
| `tweet_url` | string | Full permalink: `https://x.com/<user>/status/<id>` |
| `user_id_str` | string | Author account rest ID |
| `username` | string | Author screen name handle |

---

## 5. How to Extend x-spider

### Adding a New Exporter (e.g. SQLite or JSON Lines)
1. Create `internal/exporter/jsonl.go`.
2. Implement the `Exporter` interface defined in `internal/exporter/csv.go`:
   ```go
   type Exporter interface {
       AppendRows(rows []model.TweetRow) error
       Close() error
       GetFilePath() string
   }
   ```
3. Update `crawler.go` to instantiate your exporter when `cfg.ExportFormat == "jsonl"`.

### Adding Additional Fields
1. Open `internal/model/tweet.go`.
2. Add the field to `TweetLegacy` struct (matching Twitter GraphQL JSON key).
3. Add the field to `TweetRow` and update `ToMap()`.
4. Add the field name to `config.FilteredFields` in `internal/config/config.go`.
5. Run tests: `go test -v ./...`.

---

## 6. Testing & Quality Assurance

All core packages have dedicated unit test coverage:
- `internal/auth/auth_test.go`: Verifies AES-256 GCM encryption and key derivation.
- `internal/model/tweet_test.go`: Verifies parsing of raw Twitter GraphQL JSON payloads and date conversion.
- `internal/exporter/exporter_test.go`: Verifies CSV quoting and Excel `.xlsx` sheet creation.
- `cmd/gephi_test.go`: Verifies generation of `source, target` edge lists for network graphs.

Run all tests at any time with:
```bash
go test -v ./...
```

---

## 7. Data Engineering Guide: Curating Datasets for Machine Learning

When collecting social media data for AI training (e.g. LLM fine-tuning, sentiment analysis, instruction datasets), raw tweet streams often contain noise like crypto giveaways, bots, and link dumps.

### Pre-Filtering Strategies for Clean Data
1. **Engagement Filtering**:
   Appending `min_faves:50` or `min_retweets:15` ensures every retrieved tweet has passed community quality validation.
2. **Text Purity for Tokenization**:
   Appending `-filter:links -filter:media -filter:replies` strips out broken hyperlinks and short URLs (`t.co`), ensuring clean natural language paragraphs ideal for tokenizers and language model pre-training.
3. **Conversational Pair Mining**:
   - Use `--thread <URL>` to collect the root question and all subsequent replies.
   - Use `x-spider gephi` to map out high-centrality users and extract structured discussion trees (Prompt -> Response -> Counter-argument).
4. **Time Slicing for Volume**:
   Instead of requesting 10,000 tweets in one search, slice queries by week or month using `-f` and `--to` with `--tab LATEST`. This bypasses Twitter's pagination depth limits and collects comprehensive historical corpora.
