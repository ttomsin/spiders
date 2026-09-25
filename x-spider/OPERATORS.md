# Twitter/X Advanced Search Operators & Query Guide for `x-spider`

This guide covers all Twitter/X search operators supported by `x-spider`, how they work, and real-world examples for building datasets, monitoring topics, sentiment analysis, and AI/ML fine-tuning.

---

## Table of Contents
1. [How Search Works in `x-spider`](#how-search-works-in-x-spider)
2. [Complete Operator Reference](#complete-operator-reference)
   - [Keywords & Logic](#1-keywords--logic)
   - [Engagement & Quality Filters](#2-engagement--quality-filters)
   - [Date & Time Filters](#3-date--time-filters)
   - [People & Accounts](#4-people--accounts)
   - [Content & Media Filters](#5-content--media-filters)
   - [Language & Geolocation](#6-language--geolocation)
3. [Practical Real-World Recipes & Examples](#practical-real-world-recipes--examples)
   - [Recipe 1: High-Signal AI / Tech Insights](#recipe-1-high-signal-ai--tech-insights)
   - [Recipe 2: Historical Date Range Investigation](#recipe-2-historical-date-range-investigation)
   - [Recipe 3: Clean NLP / LLM Text Pre-Training (Zero Links/Media)](#recipe-3-clean-nlp--llm-text-pre-training-zero-linksmedia)
   - [Recipe 4: Brand Sentiment & Customer Feedback](#recipe-4-brand-sentiment--customer-feedback)
   - [Recipe 5: Q&A Pair Harvesting](#recipe-5-qa-pair-harvesting)
   - [Recipe 6: Tracking Competitor / Thought Leader Outputs](#recipe-6-tracking-competitor--thought-leader-outputs)
   - [Recipe 7: Nigerian Pidgin & Low-Resource Language Crawling](#recipe-7-nigerian-pidgin--low-resource-language-crawling)
4. [Quoting & CLI Escaping Rules](#quoting--cli-escaping-rules)

---

## How Search Works in `x-spider`

When you supply `-s` or `--search` to `x-spider`:
```powershell
.\x-spider.exe -s "golang min_faves:100" -l 200 -e csv
```

`x-spider` encodes your exact query and sends it directly to Twitter's native search engine (`https://x.com/search?q=...`). This means:
- **No artificial limitations**: Every query feature supported by Twitter web search works inside `x-spider`.
- **Server-side execution**: Filtering happens directly on Twitter's servers before results reach the spider, saving bandwidth, time, and rate limits.

---

## Complete Operator Reference

### 1. Keywords & Logic

| Operator | Example | Description |
| :--- | :--- | :--- |
| **Space (AND)** | `golang concurrency` | Matches tweets containing **both** terms anywhere in text. |
| **OR** | `react OR vue OR svelte` | Matches tweets containing **at least one** of the terms. Must be capitalized. |
| **Quotes (`""`)** | `"large language model"` | Matches the **exact phrase**. |
| **Minus (`-`)** | `crypto -airdrop -giveaway` | **Excludes** tweets containing the given keyword or hashtag. |
| **Group (`()`)** | `(python OR golang) backend` | Groups logical conditions together. |
| **Hashtags (`#`)** | `#buildinpublic` | Matches specific hashtags. |
| **Cashtags (`$`)** | `$NVDA OR $TSLA` | Matches financial ticker symbols. |

---

### 2. Engagement & Quality Filters

Filter out low-effort chatter, bot spam, and noisy unvetted posts:

| Operator | Example | Description |
| :--- | :--- | :--- |
| `min_faves:N` | `min_faves:50` | Only tweets with at least **N** likes/favorites. |
| `min_retweets:N` | `min_retweets:20` | Only tweets with at least **N** retweets. |
| `min_replies:N` | `min_replies:10` | Only tweets with at least **N** replies (great for discovering discussions). |

---

### 3. Date & Time Filters

Search historical data across specific timeframes:

| Operator | Example | Description |
| :--- | :--- | :--- |
| `since:YYYY-MM-DD` | `since:2024-01-01` | Tweets posted on or after this date. |
| `until:YYYY-MM-DD` | `until:2024-06-30` | Tweets posted up to this date (exclusive). |
| `since: & until:` | `since:2024-01-01 until:2024-02-01` | Tweets posted strictly within January 2024. |

> **Note on tabs**: For historical date queries, you usually want `--tab LATEST` (or Top) to browse chronological order.

---

### 4. People & Accounts

| Operator | Example | Description |
| :--- | :--- | :--- |
| `from:username` | `from:sama` | Tweets authored by `@sama`. |
| `to:username` | `to:OpenAI` | Public replies sent to `@OpenAI`. |
| `@username` | `@AnthropicAI` | Tweets that mention or tag `@AnthropicAI`. |
| `filter:verified` | `crypto filter:verified` | Only tweets posted by verified accounts. |
| `filter:blue_verified` | `AI filter:blue_verified` | Only tweets posted by X Premium / Blue subscribers. |

---

### 5. Content & Media Filters

Filter or exclude by media attachments and message structures:

| Operator | Example | Description |
| :--- | :--- | :--- |
| `-filter:links` | `deep learning -filter:links` | Exclude tweets with hyperlinks (useful for pure textual training data). |
| `filter:links` | `tutorial filter:links` | Only tweets containing external links or references. |
| `-filter:media` | `programming -filter:media` | Exclude tweets with images, videos, or gifs. |
| `filter:media` | `design filter:media` | Only tweets containing any media. |
| `filter:images` | `infographic filter:images` | Only tweets containing static images. |
| `filter:videos` | `interview filter:videos` | Only tweets containing native video or clips. |
| `-filter:replies` | `architecture -filter:replies` | Exclude reply posts (collects only root/original posts). |
| `filter:replies` | `"what do you think" filter:replies` | Only collect reply posts. |
| `filter:quote` | `filter:quote` | Only collect quote tweets. |
| `url:domain` | `url:github.com` | Tweets linking to a specific domain. |

---

### 6. Language & Geolocation

| Operator | Example | Description |
| :--- | :--- | :--- |
| `lang:code` | `lang:en` | Language filter by ISO 639-1 code (`en`, `es`, `fr`, `de`, `ja`, `ar`, etc.). |
| `lang:und` | `lang:und` | Undefined language (slang, heavy code-mixing, dialect, acronyms). |
| `near:location` | `near:"San Francisco"` | Tweets posted geographically near a city/area. |
| `within:radius` | `within:15mi` | Distance radius (used in conjunction with `near:`). |
| `geocode:lat,long,rad` | `geocode:37.7749,-122.4194,10km` | Strict coordinate + radius matching. |

---

## Practical Real-World Recipes & Examples

### Recipe 1: High-Signal AI / Tech Insights
Collect high-engagement insights while removing bot giveaways, airdrops, and spam:
```powershell
.\x-spider.exe -s '("machine learning" OR "deep learning") min_faves:100 min_retweets:20 lang:en -crypto -airdrop -giveaway' -l 300 -e csv -o "ai_insights"
```

---

### Recipe 2: Historical Date Range Investigation
Harvest tweets from a specific product launch window (e.g. March 2024):
```powershell
.\x-spider.exe -s '"Claude 3" since:2024-03-01 until:2024-03-15 min_faves:25' --tab LATEST -l 250 -e json -o "claude_launch"
```

---

### Recipe 3: Clean NLP / LLM Text Pre-Training (Zero Links/Media)
Harvest pure text without broken shortlinks (`t.co`), images, or reply chatter to maximize tokenizer efficiency:
```powershell
.\x-spider.exe -s '"distributed systems" -filter:links -filter:media -filter:replies min_faves:15 lang:en' -l 500 -e jsonl -o "distributed_corpus"
```

---

### Recipe 4: Brand Sentiment & Customer Feedback
Track complaints or feedback regarding an organization or product:
```powershell
.\x-spider.exe -s '("Linear" OR "Notion") ("bug" OR "broken" OR "slow" OR "issue" OR "crash") -filter:retweets lang:en' --tab LATEST -l 200 -e excel -o "product_feedback"
```

---

### Recipe 5: Q&A Pair Harvesting
Discover community questions to build Q&A and instruction-following datasets:
```powershell
.\x-spider.exe -s '("how do I" OR "what is the best way to" OR "is it possible to") "golang" ? min_faves:5 lang:en' -l 150 -e json -o "golang_qa"
```

---

### Recipe 6: Tracking Competitor / Thought Leader Outputs
Collect all standalone posts from specific creators or researchers:
```powershell
.\x-spider.exe -s '(from:karpathy OR from:ylecun OR from:sama) -filter:replies min_faves:50' -l 200 -e csv -o "ai_leaders"
```

---

### Recipe 7: Nigerian Pidgin & Low-Resource Language Crawling
Harvest code-mixed or colloquial West African English / Pidgin conversations using anchor words and `lang:und`:
```powershell
.\x-spider.exe -s '(wetin OR "dey happen" OR "no be" OR "e don" OR "abi you" OR "na so") lang:und' --tab LATEST -l 200 -e json -o "pidgin_corpus"
```

---

## Quoting & CLI Escaping Rules

When running search queries containing quotes, spaces, or parentheses via Windows PowerShell or Command Prompt, wrap the entire `-s` argument in quotes:

### In PowerShell:
Use outer double quotes and escape inner double quotes with backticks (`` ` ``), or use single quotes around the outer query:

```powershell
# Option A: Single quotes outside (Recommended for complex queries)
.\x-spider.exe -s '("large language model" OR "generative ai") min_faves:50 lang:en' -l 100 -e csv

# Option B: Double quotes with backtick escapes
.\x-spider.exe -s "(`"open source`" OR `"self-hosted`") min_faves:25 lang:en" -l 100 -e csv
```

### In Bash / Zsh (Linux & macOS):
```bash
./x-spider -s '("machine learning" OR "deep learning") min_faves:50 lang:en' -l 100 -e csv
```
