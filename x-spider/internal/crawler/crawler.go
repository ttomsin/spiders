package crawler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"x-spider/internal/auth"
	"x-spider/internal/checkpoint"
	"x-spider/internal/chunker"
	"x-spider/internal/cleaner"
	"x-spider/internal/config"
	"x-spider/internal/exporter"
	"x-spider/internal/model"
	"x-spider/internal/notifier"
	"x-spider/internal/proxy"
	"x-spider/internal/session"
)

const (
	timeoutLimit    = 20
	reachTimeoutMax = 3
)

// Crawler represents the Twitter scraping engine
type Crawler struct {
	cfg             *config.Config
	browser         *rod.Browser
	exporter        exporter.Exporter
	cleaner         *cleaner.Cleaner
	cpMgr           *checkpoint.Manager
	notif           *notifier.Notifier
	sessionStore    *session.Store
	collectedTweets []model.TweetRow
}

// NewCrawler initializes a crawler instance with configuration
func NewCrawler(cfg *config.Config) *Crawler {
	cfg.Normalize()
	var sStore *session.Store
	if cfg.SessionID != "" {
		if s, err := session.OpenStore(); err == nil {
			sStore = s
		} else {
			fmt.Printf("Warning: failed to open sessions.db: %v\n", err)
		}
	}

	return &Crawler{
		cfg: cfg,
		cleaner: cleaner.NewCleaner(cleaner.Options{
			StripURLs:     cfg.StripURLs,
			StripMentions: cfg.StripMentions,
			StripEmojis:   cfg.StripEmojis,
			MinLength:     cfg.MinLength,
		}),
		cpMgr:        checkpoint.NewManager(""),
		notif:        notifier.NewNotifier(cfg.Notifications),
		sessionStore: sStore,
	}
}

// Run executes the Twitter scraping workflow (handling chunks, resumes, and multi-tokens)
func (c *Crawler) Run(ctx context.Context) error {
	cfg := c.cfg
	cfg.Normalize()
	startTime := time.Now()

	green := color.New(color.FgGreen).SprintfFunc()
	cyan := color.New(color.FgCyan).SprintfFunc()

	if cfg.NoFile {
		c.exporter = exporter.NewMemoryExporter()
		fmt.Printf("%s\n", cyan("ℹ Running in no-file streaming mode (tweets will be dispatched via webhook)."))
	} else {
		targetFile, err := cfg.BuildTargetFilePath()
		if err != nil {
			return fmt.Errorf("failed to determine output path: %w", err)
		}

		// Initialize Exporter (CSV, Excel, JSON, JSONL, or SQLite)
		switch cfg.ExportFormat {
		case "xlsx":
			c.exporter, err = exporter.NewExcelExporter(targetFile, cfg.CSVInsertMode)
		case "json":
			c.exporter, err = exporter.NewJSONExporter(targetFile, cfg.CSVInsertMode)
		case "jsonl":
			c.exporter, err = exporter.NewJSONLExporter(targetFile, cfg.CSVInsertMode)
		case "sqlite":
			c.exporter, err = exporter.NewSQLiteExporter(targetFile, cfg.CSVInsertMode)
		default:
			c.exporter, err = exporter.NewCSVExporter(targetFile, cfg.CSVInsertMode)
		}
		if err != nil {
			return fmt.Errorf("failed to initialize exporter: %w", err)
		}
	}
	defer c.exporter.Close()

	if c.sessionStore != nil {
		defer c.sessionStore.Close()
	}

	// Check if date chunking is requested
	if cfg.Chunk != "" && cfg.FromDate != "" && cfg.ToDate != "" {
		intervals, err := chunker.SliceDateRange(cfg.FromDate, cfg.ToDate, cfg.Chunk)
		if err != nil {
			return fmt.Errorf("date chunker error: %w", err)
		}

		fmt.Printf("%s\n", cyan("Automated Date Chunker: Sliced timeframe into %d %s chunks.", len(intervals), cfg.Chunk))
		totalHarvested := 0

		for idx, interval := range intervals {
			fmt.Printf("\n%s\n", cyan("━━━ Chunk [%d/%d]: %s to %s ━━━", idx+1, len(intervals), interval.From, interval.To))
			cfg.FromDate = interval.From
			cfg.ToDate = interval.To

			count, err := c.runSingleSession(ctx)
			totalHarvested += count
			if err != nil {
				c.notifyResult("error", totalHarvested, startTime, err.Error())
				return err
			}

			if totalHarvested >= cfg.Limit {
				break
			}
		}

		_ = c.cpMgr.Clear()
		c.notifyResult("completed", totalHarvested, startTime, "")
		fmt.Printf("\n%s\n", green("✓ Finished all date chunks! Harvested %d tweets to: %s", totalHarvested, c.exporter.GetFilePath()))
		return nil
	}

	// Normal single session
	count, err := c.runSingleSession(ctx)
	if err != nil {
		c.notifyResult("error", count, startTime, err.Error())
		return err
	}

	_ = c.cpMgr.Clear()
	c.notifyResult("completed", count, startTime, "")
	return nil
}

func (c *Crawler) runSingleSession(ctx context.Context) (int, error) {
	cfg := c.cfg
	blue := color.New(color.FgBlue).SprintfFunc()
	cyan := color.New(color.FgCyan).SprintfFunc()
	yellow := color.New(color.FgYellow).SprintfFunc()
	green := color.New(color.FgGreen).SprintfFunc()
	red := color.New(color.FgRed).SprintfFunc()
	gray := color.New(color.FgHiBlack).SprintfFunc()

	// Handle Token Pool
	var tokenPool *auth.TokenPool
	activeToken := cfg.AuthToken
	if len(cfg.AuthTokens) > 0 {
		tokenPool = auth.NewTokenPool(cfg.AuthTokens)
		if t, err := tokenPool.Current(); err == nil {
			activeToken = t
		}
	}

	// Handle Proxy Pool
	activeProxy := cfg.Proxy
	if len(cfg.Proxies) > 0 {
		pPool := proxy.NewPool(cfg.Proxies)
		activeProxy = pPool.Current()
	}

	fmt.Printf("\n%s\n\n", blue("Opening Twitter search page..."))

	l := launcher.New().
		Leakless(false).
		Headless(cfg.Headless).
		Devtools(cfg.Debug)

	if activeProxy != "" {
		l.Proxy(activeProxy)
	}

	url, err := l.Launch()
	if err != nil {
		return 0, fmt.Errorf("failed to launch chromium: %w", err)
	}

	browser := rod.New().ControlURL(url)
	if err := browser.Connect(); err != nil {
		return 0, fmt.Errorf("failed to connect to browser: %w", err)
	}
	c.browser = browser

	if !cfg.Debug {
		defer browser.Close()
	}

	page, err := stealth.Page(browser)
	if err != nil {
		return 0, fmt.Errorf("failed to create stealth page: %w", err)
	}

	_ = page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:             1240,
		Height:            1080,
		DeviceScaleFactor: 1,
	})

	// Inject initial auth_token cookie
	setCookie := func(token string) error {
		return page.SetCookies([]*proto.NetworkCookieParam{
			{
				Name:     "auth_token",
				Value:    token,
				Domain:   ".x.com",
				Path:     "/",
				HTTPOnly: true,
				Secure:   true,
				SameSite: proto.NetworkCookieSameSiteStrict,
			},
		})
	}
	if err := setCookie(activeToken); err != nil {
		return 0, fmt.Errorf("failed to set auth_token cookie: %w", err)
	}

	// Setup network interception
	interceptor, err := NewNetworkInterceptor(page, cfg)
	if err != nil {
		return 0, fmt.Errorf("failed to setup network interception: %w", err)
	}
	defer interceptor.Stop()

	performCrawl := func(targetURL string) (int, bool, error) {
		fmt.Printf("%s: %s\n", blue("Navigating to"), targetURL)
		if err := page.Navigate(targetURL); err != nil {
			return 0, false, fmt.Errorf("navigation error: %w", err)
		}

		if err := page.WaitLoad(); err != nil {
			return 0, false, fmt.Errorf("page load error: %w", err)
		}

		time.Sleep(2 * time.Second)

		info, err := page.Info()
		if err == nil && strings.Contains(info.URL, "/login") {
			return 0, false, fmt.Errorf("%s", red("Invalid Twitter auth token. Please verify your auth credentials."))
		}

		isDetailMode := cfg.ThreadURL != ""
		if !isDetailMode {
			if err := InputKeywords(page, cfg); err != nil {
				fmt.Printf("%s\n", gray("Notice on keywords input: %v", err))
			}
		}

		var (
			totalSaved            int
			additionalTweetsCount int
			timeoutCount          int
			reachTimeout          int
			rateLimitCount        int
			emptyBatchCount       int
			lastTweetID           string
			seenTweetIDs          = make(map[string]bool)
			notFoundOnTab         bool
		)

		// Session Tracking & Deduplication from SQLite (~/.x-spider/sessions.db)
		if c.sessionStore != nil && cfg.SessionID != "" {
			sessionRec, err := c.sessionStore.GetOrCreateSession(cfg.SessionID, BuildSearchKeyword(cfg), cfg.SinceID)
			if err == nil && sessionRec != nil {
				if cfg.SinceID == "" && sessionRec.SinceID != "" {
					cfg.SinceID = sessionRec.SinceID
					fmt.Printf("%s\n", cyan("ℹ Active session '%s' loaded since_id: %s", cfg.SessionID, cfg.SinceID))
				}
				if historicalSeen, err := c.sessionStore.GetSeenTweetIDs(cfg.SessionID); err == nil && len(historicalSeen) > 0 {
					for id := range historicalSeen {
						seenTweetIDs[id] = true
					}
					fmt.Printf("%s\n", cyan("ℹ Active session '%s' loaded %d previously crawled tweets for deduplication", cfg.SessionID, len(historicalSeen)))
				}
			}
		}

		// Checkpoint Resume
		if cfg.Resume && c.cpMgr.Exists() {
			if st, err := c.cpMgr.Load(); err == nil && st.TotalSaved > 0 {
				totalSaved = st.TotalSaved
				lastTweetID = st.LastTweetID
				fmt.Printf("%s\n", yellow("ℹ Resuming crawl from checkpoint (%d tweets already saved)", totalSaved))
			}
		}

		for totalSaved < cfg.Limit && (timeoutCount < timeoutLimit || reachTimeout < reachTimeoutMax) {
			select {
			case <-ctx.Done():
				fmt.Println("\nCrawling interrupted by user.")
				return totalSaved, false, ctx.Err()

			case <-interceptor.RateLimitChannel():
				// If token pool has multiple accounts, rotate immediately
				if tokenPool != nil && tokenPool.Size() > 1 {
					if nextTok, err := tokenPool.Rotate(); err == nil {
						fmt.Printf("\n%s\n", yellow("ℹ Rate limit hit. Rotating to next Twitter account in pool (%s)...", auth.MaskToken(nextTok)))
						_ = setCookie(nextTok)
						time.Sleep(2 * time.Second)
						_ = page.Navigate(targetURL)
						continue
					}
				}

				backoff := CalculateForRateLimit(rateLimitCount, cfg.EnableExponentialBackoff)
				rateLimitCount++
				fmt.Printf("\n%s\n", red("Twitter rate limit encountered. Waiting %v before retrying...", backoff))
				time.Sleep(backoff)
				if btn, err := page.ElementR("button", "(?i)retry"); err == nil {
					_ = btn.Click("left", 1)
				}

			case rawJSON := <-interceptor.DataChannel():
				rateLimitCount = 0
				timeoutCount = 0

				rows, err := model.ExtractTweetsFromJSON(rawJSON, isDetailMode)
				if err != nil {
					continue
				}

				var newRows []model.TweetRow
				for _, r := range rows {
					// Apply NLP Pre-Cleaner Pipeline
					cleaned, pass := c.cleaner.Clean(r)
					if !pass {
						continue
					}

					if !seenTweetIDs[cleaned.IDStr] {
						// Filter by SinceID if specified
						if cfg.SinceID != "" && cleaned.IDStr <= cfg.SinceID {
							continue
						}

						seenTweetIDs[cleaned.IDStr] = true
						newRows = append(newRows, *cleaned)
						lastTweetID = cleaned.IDStr
					}
				}

				if len(newRows) > 0 {
					emptyBatchCount = 0

					if totalSaved+len(newRows) > cfg.Limit {
						remaining := cfg.Limit - totalSaved
						newRows = newRows[:remaining]
					}

					if err := c.exporter.AppendRows(newRows); err != nil {
						fmt.Printf("%s\n", red("Error saving rows: %v", err))
					} else {
						totalSaved += len(newRows)
						additionalTweetsCount += len(newRows)
						c.collectedTweets = append(c.collectedTweets, newRows...)

						// Record to SQLite SessionStore if session-id is active
						if c.sessionStore != nil && cfg.SessionID != "" {
							var ids []string
							for _, nr := range newRows {
								ids = append(ids, nr.IDStr)
							}
							_ = c.sessionStore.RecordTweets(cfg.SessionID, ids, lastTweetID)
						}

						// Save checkpoint if not in ephemeral mode
						if !cfg.NoFile {
							_ = c.cpMgr.Save(checkpoint.State{
								Query:        BuildSearchKeyword(cfg),
								TargetFile:   c.exporter.GetFilePath(),
								ExportFormat: cfg.ExportFormat,
								LastTweetID:  lastTweetID,
								TotalSaved:   totalSaved,
								TargetLimit:  cfg.Limit,
							})
							fmt.Printf("\n\n%s\n", blue("Your tweets saved to: %s", c.exporter.GetFilePath()))
						} else {
							fmt.Printf("\n\n%s\n", cyan("Tweets collected in memory for webhook dispatch."))
						}
						fmt.Printf("%s\n", yellow("Total tweets saved: %d / %d", totalSaved, cfg.Limit))

						if additionalTweetsCount > 100 {
							additionalTweetsCount = 0
							if cfg.DelayEvery100Tweets > 0 {
								fmt.Printf("%s\n", gray("-- Taking a break, waiting for %d seconds...", cfg.DelayEvery100Tweets))
								time.Sleep(time.Duration(cfg.DelayEvery100Tweets) * time.Second)
							}
						} else if additionalTweetsCount > 20 {
							time.Sleep(time.Duration(cfg.DelaySeconds) * time.Second)
						}
					}
				} else {
					emptyBatchCount++
					if emptyBatchCount >= 4 {
						fmt.Printf("\n%s\n", yellow("No more tweets available for this search criteria."))
						break
					}
				}

				_ = ScrollDown(page)

			case <-time.After(1500 * time.Millisecond):
				timeoutCount++
				if timeoutCount == 1 {
					fmt.Printf("%s", gray("\n-- Scrolling... (%d)", timeoutCount))
				} else {
					fmt.Printf("%s", gray(" (%d)", timeoutCount))
				}

				if timeoutCount > timeoutLimit {
					if reachTimeout < reachTimeoutMax {
						reachTimeout++
						fmt.Printf("\n%s\n", yellow("Timeout reached %d times, verifying again...", reachTimeout))
						timeoutCount = 0
						_ = ScrollUp(page)
						time.Sleep(2 * time.Second)
						_ = ScrollDown(page)
					} else {
						if el, err := page.ElementR("*", "(?i)no results for"); err == nil && el != nil {
							notFoundOnTab = true
						}
						break
					}
				}

				_ = ScrollDown(page)
			}
		}

		return totalSaved, notFoundOnTab, nil
	}

	initialURL := cfg.GetSearchURL()
	totalSaved, notFound, err := performCrawl(initialURL)
	if err != nil {
		c.captureErrorScreenshot(page)
		return totalSaved, err
	}

	if totalSaved == 0 && notFound && cfg.ThreadURL == "" {
		switchedTab, switchedURL := cfg.GetSwitchedSearchURL()
		fmt.Printf("\n%s\n", yellow("No tweets found on '%s' tab, trying '%s' tab...", cfg.SearchTab, switchedTab))
		_, _, err = performCrawl(switchedURL)
		if err != nil {
			c.captureErrorScreenshot(page)
			return totalSaved, err
		}
	}

	if totalSaved > 0 {
		fmt.Printf("\n%s\n", green("Successfully crawled %d tweets! Output saved to: %s", totalSaved, c.exporter.GetFilePath()))
	} else {
		fmt.Printf("\n%s\n", yellow("No tweets found for the given search criteria."))
	}

	return totalSaved, nil
}

func (c *Crawler) notifyResult(status string, count int, start time.Time, errMsg string) {
	if c.notif == nil || !c.notif.Enabled() {
		return
	}
	durationStr := time.Since(start).Round(time.Second).String()
	filePath := ""
	if c.exporter != nil && !c.cfg.NoFile {
		filePath = c.exporter.GetFilePath()
	}

	payload := notifier.Payload{
		Status:      status,
		Query:       c.cfg.SearchKeyword,
		TweetsSaved: count,
		OutputFile:  filePath,
		Duration:    durationStr,
		Error:       errMsg,
	}

	if c.notif.IncludeData() && len(c.collectedTweets) > 0 {
		payload.Data = c.collectedTweets
	}

	_ = c.notif.Notify(payload)
}

func (c *Crawler) captureErrorScreenshot(page *rod.Page) {
	if page == nil {
		return
	}
	nowStr := time.Now().Format("02-01-2006 15-04-05")
	nowStr = strings.ReplaceAll(nowStr, " ", "_")
	nowStr = strings.ReplaceAll(nowStr, ":", "-")

	errFilename := filepath.Join(c.cfg.FolderDestination, fmt.Sprintf("Error-%s.png", nowStr))
	_ = os.MkdirAll(c.cfg.FolderDestination, 0755)

	if buf, err := page.Screenshot(true, nil); err == nil {
		_ = os.WriteFile(errFilename, buf, 0644)
		red := color.New(color.FgRed).SprintfFunc()
		fmt.Printf("\n%s\n", red("An error occurred. A screenshot was saved to: %s", errFilename))
	}
}
