# x-spider

```text
      / _ \      __  __           ____        _     _           
    \_\(_)/_/    \ \/ /          / ___| _ __ (_) __| | ___ _ __ 
     _//o\\_      \  /  _____   \___ \| '_ \| |/ _` |/ _ \ '__|
      /   \       /  \ |_____|   ___) | |_) | | (_| |  __/ |   
     /     \     /_/\_\         |____/| .__/|_|\__,_|\___|_|   
                                      |_|                      
               [Advanced Twitter/X Intelligence Crawler]
```

**x-spider** is a high-performance, native Go crawler designed to search and extract tweets, user metadata, and conversation threads from Twitter/X. Built on native Chrome DevTools Protocol (CDP) automation, it intercepts network data directly from Twitter's internal GraphQL stream for speed, stealth, and reliability.

---

## Highlights

- 🚀 **Zero External Dependencies**: Compiled into a single standalone binary. No Node.js or npm runtime required.
- 🔐 **Persistent Encrypted Authentication**: Save your Twitter auth token once (`x-spider auth set-token`), and never pass it again. Stored locally with AES-256 GCM encryption.
- 🕵️ **Built-in Bot Evasion**: Full stealth anti-detection measures (`go-rod/stealth`) to bypass automated bot checks.
- ⚡ **Bandwidth Saving Media Blocker**: Drops unnecessary image, video, and font downloads to speed up scrolling and save RAM.
- 🌐 **Direct GraphQL Interception**: Captures raw `SearchTimeline` and `TweetDetail` payloads straight from HTTP responses.
- ⚙️ **Configurable via YAML**: Customize search queries, filters, delays, and exports in `config.yaml`.
- 📊 **Multi-Format Export**: Generates standardized, alphabetically-sorted output in RFC-4180 **CSV**, **Excel (.xlsx)**, or formatted **JSON**.
- 🕸️ **Gephi Network Exporter**: Subcommand to convert reply threads into `source, target` edge lists for graph analysis.

---

## Installation & Build

Requires Go 1.26 or higher:

```bash
# Clone or open the x-spider directory
cd x-spider

# Build the executable
go build -o x-spider.exe .
```

---

## Quick Start & Authentication

### 1. One-Time Authentication Setup
Extract your `auth_token` cookie from [x.com](https://x.com) via your browser's Developer Tools (under **Application > Storage > Cookies > https://x.com**), then save it:

```bash
.\x-spider.exe auth set-token
# Enter your token securely when prompted (input is hidden)
```

Check your stored auth status anytime:
```bash
.\x-spider.exe auth status
```

You can now run searches without passing the token flag or editing config files!

### 2. Basic Search (Export to CSV)
```bash
.\x-spider.exe -s "golang" -l 25
```

### 3. Export to JSON Lines (.jsonl) for AI/ML Training
```bash
.\x-spider.exe -s "deep learning" -l 100 -e jsonl
```

### 4. Export Directly to Local SQLite Database (.db)
```bash
.\x-spider.exe -s "cybersecurity" -l 200 -e sqlite
```

### 5. Automated Date Chunker (Scrape Year-by-Month)
```bash
.\x-spider.exe -s "crypto" -f "01-01-2025" --to "31-12-2025" --chunk monthly --tab LATEST -l 2000
```

### 6. On-the-Fly NLP Pre-Cleaning (Clean Text for LLM Datasets)
```bash
.\x-spider.exe -s "machine learning" --strip-urls --strip-mentions --strip-emojis --min-length 30 -e jsonl
```

### 7. Fault-Tolerant Resume from Interruption
```bash
.\x-spider.exe --resume
```

### 8. Crawl a Specific Discussion Thread
```bash
.\x-spider.exe --thread "https://x.com/username/status/1234567890" -l 50
```

---

## Configuration File (`config.yaml`)

`x-spider` automatically loads `config.yaml` from the working directory if present, or you can supply custom files via `-c <path>`.

See [`config.example.yaml`](config.example.yaml) for a complete template:

```yaml
# Search Query Options
search_keyword: "golang"
from_date: "01-01-2026"  # DD-MM-YYYY
to_date: "01-02-2026"    # DD-MM-YYYY
search_tab: "TOP"        # TOP or LATEST

# Limits & Delays
limit: 50
delay_seconds: 3
delay_every_100_tweets: 10

# Export Configuration
export_format: "csv"       # csv or xlsx
csv_insert_mode: "REPLACE" # REPLACE or APPEND
output_filename: ""        # Defaults to '<keyword> <timestamp>'
folder_destination: "./tweets-data"

# Browser & Media Settings
headless: true
debug: false
enable_exponential_backoff: false
```

Precedence order: **CLI Flags** > **Environment Variables** > **config.yaml** > **Saved Credentials** > **Interactive Prompts** > **Defaults**.

---

## Command Line Reference

```bash
x-spider [flags]
x-spider [command]
```

### Available Commands

| Command | Description | When to Use |
|---------|-------------|-------------|
| `crawl` | Runs a crawl operation (default action) | When specifying search keywords or threads |
| `auth` | Manage saved Twitter credentials | To save, check, or clear persistent authentication |
| `gephi` | Transform scraped CSV into network edge list | To visualize conversations in Gephi / NetworkX |
| `help` | Help about any command | To read detailed parameter information |

### Auth Commands (Multi-Account Token Pool)

Manage multiple Twitter accounts securely stored on your machine. When multiple accounts are configured, `x-spider` automatically rotates through them whenever Twitter rate limits are reached:

```bash
# Set one or more tokens (replaces current pool)
x-spider auth set-token <token1> [token2] [token3]...

# Append a new account to your existing pool
x-spider auth add-token <new_token>

# View all stored accounts in the rotation pool
x-spider auth status

# Remove an account by index (e.g. account #2)
x-spider auth remove 2

# Clear all stored credentials
x-spider auth clear
```

### Flags

| Flag | Shorthand | Default | Description |
|------|-----------|---------|-------------|
| `--config` | `-c` | `""` | Path to YAML configuration file |
| `--token` | `-t` | `""` | Twitter auth token (not needed if saved via `auth set-token`) |
| `--search-keyword` | `-s` | `""` | Search query or keyword |
| `--thread` | | `""` | Tweet thread URL for discussion scraping |
| `--from` | `-f` | `""` | Start date in `DD-MM-YYYY` format |
| `--to` | | `""` | End date in `DD-MM-YYYY` format |
| `--chunk` | | `""` | Automated date slicing: `monthly`, `weekly`, or `daily` |
| `--resume` | | `false` | Resume crawl from `.xspider-checkpoint.json` |
| `--limit` | `-l` | `10` | Maximum number of tweets to collect |
| `--delay` | `-d` | `3` | Delay between tweet batch requests (seconds) |
| `--tab` | | `"TOP"` | Search tab: `TOP` or `LATEST` |
| `--export-format` | `-e` | `"csv"` | Output format: `csv`, `xlsx`, `json`, `jsonl`, or `sqlite` |
| `--output-filename` | `-o` | `""` | Custom output filename (without extension) |
| `--strip-urls` | | `false` | Remove hyperlinks from tweet text (NLP pre-cleaning) |
| `--strip-mentions` | | `false` | Remove `@username` tags from tweet text |
| `--strip-emojis` | | `false` | Remove Unicode emojis and pictographs |
| `--min-length` | | `0` | Discard tweets with clean character length below threshold |
| `--headless` | | `true` | Run Chromium in headless mode |
| `--debug` | | `false` | Keep browser window open on error |
| `--backoff` | | `false` | Enable exponential backoff on rate limits |

---

## Tips for Maximum Yield

### 1. Use the `LATEST` Tab Instead of `TOP`
The `TOP` tab only shows curated high-engagement tweets, which can be as few as 10–20 tweets for specific queries or date ranges. The `LATEST` tab returns the chronological live stream with vastly more data:

```powershell
.\x-spider.exe -s "machine learning" -f "01-01-2026" --to "01-02-2026" --tab LATEST -l 100
```

### 2. Add a Slightly Longer Delay
When scraping large volumes (>100 tweets), use `-d 4` or `-d 5` (4–5 seconds) to mimic human browsing intervals and prevent Twitter's server-side throttling:

```powershell
.\x-spider.exe -s "machine learning" -l 200 -d 5
```

### 3. Date Slicing for Massive Datasets
Twitter searches typically cap visible results at roughly 500–800 tweets per query. To harvest tens of thousands of tweets, divide your target timeframe into monthly or weekly slices:

```powershell
# January
.\x-spider.exe -s "machine learning" -f "01-01-2026" --to "31-01-2026" --tab LATEST -l 500 -o "ml_jan"
# February
.\x-spider.exe -s "machine learning" -f "01-02-2026" --to "28-02-2026" --tab LATEST -l 500 -o "ml_feb"
```

---

## Advanced Search Guide: Harvesting High-Quality Datasets for AI/ML

> 📖 **Full Reference & Recipes**: See [OPERATORS.md](OPERATORS.md) for the complete list of Twitter/X search operators, date ranges, media filters, and domain-specific search recipes.

If you are gathering data to **fine-tune LLMs, train sentiment classifiers, create QA pairs, or build domain-specific NLP models**, raw social media text can be full of spam, bots, and noise.

You can combine Twitter's advanced search operators directly inside the `-s` keyword parameter to curate clean, high-signal training datasets.

### 1. High-Signal Quality Filtering (Filter Spam & Low-Effort Posts)
Use engagement thresholds to discard bots, spam accounts, and zero-engagement chatter:

- `min_faves:N`: Only tweets with at least *N* likes.
- `min_retweets:N`: Only tweets with at least *N* retweets.
- `min_replies:N`: Great for finding topics that spark deep discussions.

```powershell
# Collect community-vetted, high-quality AI insights
.\x-spider.exe -s "machine learning min_faves:50 min_retweets:10 lang:en" -l 200
```

### 2. Pure Natural Language for LLM Pre-Training (No Broken URLs)
URLs, shortlinks (`t.co`), and media tags degrade tokenizer efficiency and LLM perplexity. Strip them out at the search level:

- `-filter:links`: Excludes tweets containing hyperlinks.
- `-filter:media`: Excludes tweets with images or videos.
- `-filter:replies`: Only collect original standalone posts.

```powershell
# Pure English sentences on deep learning without links or media
.\x-spider.exe -s "deep learning -filter:links -filter:media -filter:replies lang:en min_faves:20" -l 300
```

### 3. Negative Keyword Filtering (Anti-Spam & Anti-Airdrop)
Certain keywords attract massive bot engagement (crypto airdrops, giveaways, NFT promotions). Filter them out with negative keywords:

```powershell
.\x-spider.exe -s '"artificial intelligence" -crypto -airdrop -giveaway -nft -bot lang:en min_faves:25' -l 200
```

### 4. Harvesting Q&A & Instruction-Tuning Datasets
To build instruction/response datasets:
- **Questions**: Search for tweets ending with question marks or starting with question words:
  ```powershell
  .\x-spider.exe -s '"how do I" OR "what is the best" "machine learning" ? min_faves:10 lang:en' -l 150
  ```
- **Discussion Threads**: When someone shares a detailed tutorial or breakdown, grab the thread URL and crawl all author posts and community replies:
  ```powershell
  .\x-spider.exe --thread "https://x.com/karpathy/status/1234567890" -l 100 -e csv
  ```

### 5. Domain Expert & Influencer Harvesting
To build expert datasets from leading researchers or authoritative sources:

```powershell
# Collect high-engagement posts from specific researchers or organizations
.\x-spider.exe -s "from:ylecun OR from:AndrewYNg min_faves:50" -l 200
```

### 6. Languages Supported on Twitter/X & Harvesting Dialects (Pidgin, Slang, `lang:und`)

Twitter automatically tags each tweet with an ISO 639-1 language code. You can filter for any supported language using the `lang:<code>` operator in your search query:

#### Supported Language Codes

| Code | Language | Code | Language | Code | Language |
|------|----------|------|----------|------|----------|
| `en` | English | `es` | Spanish | `fr` | French |
| `de` | German | `it` | Italian | `pt` | Portuguese |
| `ar` | Arabic | `ja` | Japanese | `ko` | Korean |
| `zh` | Chinese | `ru` | Russian | `hi` | Hindi |
| `id` | Indonesian | `tr` | Turkish | `fa` | Persian |
| `sw` | Swahili | `yo` | Yoruba | `ig` | Igbo |
| `ha` | Hausa | `nl` | Dutch | `und` | **Undefined** (slang/code-mix) |

#### What is `lang:und` (Undefined Language)?
Twitter assigns `lang:und` to tweets that its language classifier **cannot identify as a single standard language with high confidence**. This includes:
- Heavy street slang, unstandardized dialects, and short colloquial phrases.
- Code-mixed / multilingual sentences (e.g. mixing English with regional tongues).
- Tweets dominated by emojis, acronyms, or hashtags.

**How to use `lang:und`:**
```powershell
# Collect dialect, heavy slang, or code-mixed tweets
.\x-spider.exe -s '(wetin OR dey OR "how far") lang:und' --tab LATEST -l 100 -e json
```

#### Does Twitter Support Nigerian Pidgin (`lang:pcm`)?
**No**, Twitter does not officially index Nigerian Pidgin under `lang:pcm` (searching `lang:pcm` returns 0 results). Twitter's model typically tags Pidgin as `en` (due to shared vocabulary) or `und`.

To harvest high-quality Nigerian Pidgin datasets (as done in *NaijaSenti* and *AfriSenti* NLP benchmarks), use these 3 strategies:

1. **High-Frequency Pidgin Anchor Words (Highest Yield)**:
   Search for distinctive grammatical markers (`wetin`, `dey`, `wey`, `na`, `fit`, `don`, `sef`, `abi`, `una`):
   ```powershell
   .\x-spider.exe -s '(wetin OR "dey happen" OR "no be" OR "e don" OR "abi you" OR "na so")' --tab LATEST -l 200 -e json
   ```
2. **Harvest from Verified Pidgin Outlets (Gold-Standard Corpus Quality)**:
   ```powershell
   # BBC News Pidgin
   .\x-spider.exe -s "from:bbcnewspidgin" -l 500 -e json -o "bbc_pidgin"

   # Wazobia FM (Nigeria's leading Pidgin radio network)
   .\x-spider.exe -s "from:Wazobia_FM" -l 300 -e json -o "wazobia_pidgin"
   ```
3. **Filter by Author Location in the Output**:
   `x-spider` automatically records the author's **`location`** in your exported CSV/JSON/Excel file. You can search for Nigerian cultural topics (e.g. `naira`, `lagos`, `afrobeats`) and filter by `location` (`Lagos, Nigeria`, `Abuja`, etc.).

### Quick Operator Cheat Sheet for `-s`

| Operator | Syntax Example | Use Case |
|----------|----------------|----------|
| **Language** | `lang:en` / `lang:id` / `lang:fr` | Enforce clean single-language training corpus |
| **Minimum Likes** | `min_faves:50` | Filter out spam and low-effort posts |
| **Minimum Retweets** | `min_retweets:15` | Capture viral, community-validated ideas |
| **No Links** | `-filter:links` | Clean sentences without broken `t.co` URLs |
| **No Media** | `-filter:media` | Focus purely on textual knowledge |
| **No Replies** | `-filter:replies` | Collect only original standalone posts |
| **Only Questions** | `?` (e.g. `"how to" ?`) | Build conversational Q&A training datasets |
| **Exact Match** | `"generative AI"` | Match the exact phrase |
| **Boolean OR** | `PyTorch OR TensorFlow` | Match either technology |
| **Negative Words** | `-crypto -giveaway` | Exclude spam niches |
| **From User** | `from:karpathy` | Collect thoughts from specific thought-leaders |

---

## Webhook Streaming & Direct Backend Ingestion

You can stream crawled tweets directly to your backend API via HTTP POST without needing to store or manage local files:

```bash
# Stream tweets directly to your backend without saving files to disk
.\x-spider.exe -s "machine learning" -l 100 --webhook-url "https://api.yourdomain.com/v1/tweets/ingest" --webhook-data --no-file
```

### Webhook JSON Payload Schema

When `--webhook-data` (or `webhook_data: true`) is enabled, the HTTP POST payload sent to your endpoint contains:

```json
{
  "status": "completed",
  "query": "machine learning",
  "tweets_saved": 100,
  "duration": "45s",
  "data": [
    {
      "id_str": "189674829102938475",
      "conversation_id_str": "189674829102938475",
      "username": "karpathy",
      "full_text": "Recent progress in reinforcement learning...",
      "created_at": "2026-09-06T14:30:00.000Z",
      "reply_count": 54,
      "retweet_count": 312,
      "favorite_count": 2180,
      "lang": "en",
      "tweet_url": "https://x.com/karpathy/status/189674829102938475",
      "location": "San Francisco, CA"
    }
  ]
}
```

### Stateful Resume & Deduplication (`--session-id` & `--since-id`)

When running automated pipelines or streaming directly to your backend, you can pass a persistent `--session-id`:

```bash
# First crawl: creates session tracking in ~/.x-spider/sessions.db
.\x-spider.exe -s "machine learning" --session-id "job_ml_01" --webhook-url "https://api.yourdomain.com/ingest" --webhook-data --no-file

# Subsequent crawl: automatically skips previously crawled tweets and streams ONLY new content
.\x-spider.exe -s "machine learning" --session-id "job_ml_01" --webhook-url "https://api.yourdomain.com/ingest" --webhook-data --no-file
```

`x-spider` automatically stores seen tweet IDs and the session's `since_id` inside `~/.x-spider/sessions.db` (SQLite). You can also pass `--since-id <ID>` manually to enforce a lower-bound ID cutoff.

---

## Gephi Network Conversion

Convert reply threads from your scraped data into a source-to-target edge list:

```bash
.\x-spider.exe gephi -i ./tweets-data/scraped.csv -o ./tweets-data/network_edges.csv
```

Outputs a clean CSV with `source` and `target` columns ready for network analysis.

---

## Standardized Output Schema

Scraped data is saved to `./tweets-data/` with the following 15 fields sorted alphabetically:

1. `conversation_id_str`: Thread identifier
2. `created_at`: ISO 8601 UTC timestamp (`YYYY-MM-DDTHH:MM:SS.000Z`)
3. `favorite_count`: Number of likes
4. `full_text`: Tweet text content
5. `id_str`: Tweet rest ID
6. `image_url`: Attached image URL (if present)
7. `in_reply_to_screen_name`: Username being replied to
8. `lang`: Language code
9. `location`: User location profile field
10. `quote_count`: Number of quote tweets
11. `reply_count`: Number of replies
12. `retweet_count`: Number of retweets
13. `tweet_url`: Direct tweet permalink
14. `user_id_str`: Author user rest ID
15. `username`: Author screen name handle

---

## License

MIT License. See [LICENSE](LICENSE) for details.
