# Spiders Monorepo 🕷️

```text
      / _ \                 ____        _     _               
    \_\(_)/_/              / ___| _ __ (_) __| | ___ _ __ ___ 
     _//o\\_     _____     \___ \| '_ \| |/ _` |/ _ \ '__/ __|
      /   \     |_____|     ___) | |_) | | (_| |  __/ |  \__ \
     /     \               |____/| .__/|_|\__,_|\___|_|  |___/
                                 |_|                          
                 [Next-Gen Native Web Automation & Intelligence Suite]
```

A monorepo collection of high-performance, native Go scrapers, crawlers, and automation engines. Built on the **Chrome DevTools Protocol (CDP)** using `go-rod` with automated stealth evasion, real-time data streaming, encrypted credential vaults, and zero external webdriver dependencies.

---

## 📦 Packages in this Repository

| Project | Description | Primary Capabilities | Status |
| :--- | :--- | :--- | :---: |
| **[`chatgpt-spider`](./chatgpt-spider)** | Native ChatGPT automation engine & OpenAI-compatible REST server | • OpenAI-compatible REST API (`/v1/chat/completions`)<br>• Real-time typewriter token streaming (SSE)<br>• Native Temporary Chat mode (stateless, zero clutter)<br>• Massive document pill attachments (`-F / --file`)<br>• Strict structured output enforcement (`--format json/csv`) | 🟢 **Active / v1.0.0** |
| **[`github.com/ttomsin/spiders/x-spider`](./x-spider)** | Advanced Twitter / X intelligence crawler & graph extractor | • Direct GraphQL payload interception<br>• Media blocking for fast headless crawling<br>• Search timeline & conversation thread extraction<br>• Standardized CSV, Excel (.xlsx), and JSON exports<br>• Gephi network graph edge generation (`source, target`) | 🟢 **Active / v1.0.0** |

---

## 🌟 Core Architecture & Design Principles

All spider projects in this repository share a common technical philosophy:

1. **Zero External Drivers**:
   - No `chromedriver.exe`, `geckodriver`, or Selenium grid required.
   - Communicates directly with Chromium over WebSocket via native Chrome DevTools Protocol (CDP).

2. **Stealth & Bot Evasion**:
   - Automated evasion of Cloudflare Turnstile, Akamai, and browser fingerprinting.
   - Hides `navigator.webdriver`, spoofs realistic hardware concurrency, audio contexts, and WebGL parameters.
   - Simulates humanized typing and mouse events to trigger synthetic frontend dispatchers.

3. **Production Concurrency & Windows Resilience**:
   - Pre-launch sanitization of stale Windows profile locks (`Lockfile`, `SingletonLock`) to prevent Error Code 32 sharing violations.
   - Decoupled concurrency (`sync.Mutex` for browser actions, `sync.RWMutex` for state) preventing deadlock during live streaming.
   - Graceful signal traps (`SIGINT`, `SIGTERM`) ensuring complete tree cleanup of Chromium child helper processes.

4. **Secure Credential Vaults**:
   - Session tokens and auth cookies stored locally using AES-256 GCM encryption derived from machine-specific hardware keys.

---

## 🚀 Quick Navigation

- **ChatGPT Automation & REST Daemon**:
  Read [`chatgpt-spider/README.md`](./chatgpt-spider/README.md) for CLI commands, REST endpoints, Python SDK integrations, and documentation.
  - Deep technical dive: [`chatgpt-spider/HOW-CHATGPT-SPIDER-WORKS.md`](./chatgpt-spider/HOW-CHATGPT-SPIDER-WORKS.md)
  - Real-world capabilities & use cases: [`chatgpt-spider/WHAT-YOU-CAN-USE-CHATGPT-SPIDER.md`](./chatgpt-spider/WHAT-YOU-CAN-USE-CHATGPT-SPIDER.md)

- **Twitter / X Scraping & Intelligence**:
  Read [`x-spider/README.md`](./x-spider/README.md) for keyword search configurations, GraphQL filters, and Gephi network analysis.

---

## 🛠️ Building Projects

Each spider is an independent Go module and can be compiled into a standalone binary:

```powershell
# Build chatgpt-spider
cd chatgpt-spider
go build -o chatgpt-spider.exe .

# Build x-spider
cd ../x-spider
go build -o x-spider.exe .
```

---

## 📄 License
MIT License. Created for native, high-performance web intelligence and automation.
