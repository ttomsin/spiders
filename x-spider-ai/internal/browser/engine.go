package browser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"x-spider-ai/internal/parser"
	"x-spider-ai/internal/models"
)

// Engine manages the browser lifecycle, stealth, hijacking, and DOM interactions
type Engine struct {
	browser        *rod.Browser
	page           *rod.Page
	router         *rod.HijackRouter
	sessionMgr     *SessionManager
	sessionData    *SessionData
	headless       bool
	proxyURL       string
	mu             sync.RWMutex
	capturedTweets      []models.Tweet
	onTweetChan         chan models.Tweet
	bearerToken         string
	csrfToken           string
	lastCreatedTweetID  string
	capturedUser        *models.UserProfile
}

// Config configures the browser engine
type Config struct {
	AccountID   string
	Headless    bool
	ProxyURL    string
	SessionPath string
	UserDataDir string
}

// NewEngine initializes a new stealth browser engine
func NewEngine(cfg Config) (*Engine, error) {
	sm := NewSessionManager(cfg.SessionPath)

	l := launcher.New().
		Leakless(false).
		Headless(cfg.Headless).
		Set("disable-blink-features", "AutomationControlled")

	if cfg.ProxyURL != "" {
		l.Proxy(cfg.ProxyURL)
	}
	if cfg.UserDataDir != "" {
		l.UserDataDir(cfg.UserDataDir)
	}

	u, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(u).MustConnect()

	eng := &Engine{
		browser:     browser,
		sessionMgr:  sm,
		headless:    cfg.Headless,
		proxyURL:    cfg.ProxyURL,
		onTweetChan: make(chan models.Tweet, 200),
	}

	// Try loading target or active saved session
	if sess, err := sm.LoadAccountSession(cfg.AccountID); err == nil {
		eng.sessionData = sess
		eng.csrfToken = sess.CT0
		eng.bearerToken = sess.BearerToken
	}

	return eng, nil
}

// InitPage creates the main stealth page and configures network hijacking
func (e *Engine) InitPage() error {
	page, err := stealth.Page(e.browser)
	if err != nil {
		return fmt.Errorf("failed to create stealth page: %w", err)
	}
	e.page = page

	// Inject cookies if session exists
	if e.sessionData != nil && e.sessionData.AuthToken != "" {
		if err := e.injectCookies(e.sessionData.AuthToken, e.sessionData.CT0); err != nil {
			return err
		}
	}

	// Setup Network Hijacking
	if err := e.setupHijacking(); err != nil {
		return err
	}

	return nil
}

func (e *Engine) injectCookies(authToken, ct0 string) error {
	now := proto.TimeSinceEpoch(time.Now().Add(365 * 24 * time.Hour).Unix())
	cookies := []*proto.NetworkCookieParam{
		{
			Name:     "auth_token",
			Value:    authToken,
			Domain:   ".x.com",
			Path:     "/",
			Secure:   true,
			HTTPOnly: true,
			Expires:  now,
		},
	}
	if ct0 != "" {
		cookies = append(cookies, &proto.NetworkCookieParam{
			Name:     "ct0",
			Value:    ct0,
			Domain:   ".x.com",
			Path:     "/",
			Secure:   true,
			HTTPOnly: false,
			Expires:  now,
		})
	}

	return proto.NetworkSetCookies{Cookies: cookies}.Call(e.page)
}

func (e *Engine) setupHijacking() error {
	router := e.page.HijackRequests()
	e.router = router

	router.MustAdd("*", func(ctx *rod.Hijack) {
		req := ctx.Request
		reqURL := req.URL().String()

		// Intercept Authorization & CSRF headers for headless/direct re-use
		headers := req.Headers()
		if authHdr, ok := headers["authorization"]; ok {
			e.mu.Lock()
			e.bearerToken = authHdr.String()
			e.mu.Unlock()
		}
		if csrfHdr, ok := headers["x-csrf-token"]; ok {
			e.mu.Lock()
			e.csrfToken = csrfHdr.String()
			e.mu.Unlock()
		}

		// Intercept CreateTweet response to extract the new tweet ID
		if strings.Contains(reqURL, "CreateTweet") || strings.Contains(reqURL, "CreateNoteTweet") {
			if err := ctx.LoadResponse(http.DefaultClient, true); err == nil {
				body := ctx.Response.Body()
				var payload map[string]any
				if err := json.Unmarshal([]byte(body), &payload); err == nil {
					tweets := parser.ExtractTweetsFromGraphQL([]byte(body))
					if len(tweets) > 0 {
						e.mu.Lock()
						e.lastCreatedTweetID = tweets[0].ID
						e.mu.Unlock()
					}
				}
			}
			return
		}

		// Intercept User profile GraphQL payloads
		if strings.Contains(reqURL, "UserByScreenName") || strings.Contains(reqURL, "UserByRestId") || strings.Contains(reqURL, "Viewer") {
			if err := ctx.LoadResponse(http.DefaultClient, true); err == nil {
				body := ctx.Response.Body()
				if u := parser.ExtractUserFromGraphQL([]byte(body)); u != nil {
					e.mu.Lock()
					e.capturedUser = u
					e.mu.Unlock()
				}
			}
			return
		}

		// Intercept timeline, tweets, replies, search, and details
		isTweetPayload := strings.Contains(reqURL, "SearchTimeline") ||
			strings.Contains(reqURL, "TweetDetail") ||
			strings.Contains(reqURL, "UserTweets") ||
			strings.Contains(reqURL, "HomeTimeline") ||
			strings.Contains(reqURL, "ConversationTimeline")

		if isTweetPayload {
			if err := ctx.LoadResponse(http.DefaultClient, true); err == nil {
				body := ctx.Response.Body()
				tweets := parser.ExtractTweetsFromGraphQL([]byte(body))
				if len(tweets) > 0 {
					e.mu.Lock()
					for _, tw := range tweets {
						e.capturedTweets = append(e.capturedTweets, tw)
						select {
						case e.onTweetChan <- tw:
						default:
						}
					}
					e.mu.Unlock()
				}
			}
			return
		}

		// Continue other requests
		ctx.ContinueRequest(&proto.FetchContinueRequest{})
	})

	go router.Run()
	return nil
}

// LoginWithTokens sets authentication cookies and validates login on x.com for a specific account
func (e *Engine) LoginWithTokens(accountID, screenName, authToken, ct0 string) error {
	if accountID == "" {
		accountID = "default"
	}
	e.sessionData = &SessionData{
		AccountID:   accountID,
		ScreenName:  screenName,
		AuthToken:   authToken,
		CT0:         ct0,
		CookieMap:   map[string]string{"auth_token": authToken, "ct0": ct0},
	}

	if e.page == nil {
		if err := e.InitPage(); err != nil {
			return err
		}
	} else {
		if err := e.injectCookies(authToken, ct0); err != nil {
			return err
		}
	}

	if err := e.sessionMgr.SaveSession(e.sessionData); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	// Navigate to home to confirm session
	if err := e.page.Navigate("https://x.com/home"); err != nil {
		return err
	}
	_ = e.page.Timeout(10 * time.Second).WaitLoad()
	time.Sleep(2 * time.Second)

	return nil
}

// SwitchAccount dynamically loads another account's session cookies into the running browser
func (e *Engine) SwitchAccount(accountID string) error {
	sess, err := e.sessionMgr.LoadAccountSession(accountID)
	if err != nil {
		return err
	}
	e.sessionData = sess
	e.csrfToken = sess.CT0
	e.bearerToken = sess.BearerToken

	if err := e.sessionMgr.SwitchAccount(accountID); err != nil {
		return err
	}

	if e.page != nil && sess.AuthToken != "" {
		if err := e.injectCookies(sess.AuthToken, sess.CT0); err != nil {
			return err
		}
	}
	return nil
}

// ListAccounts returns all registered accounts in SQLite
func (e *Engine) ListAccounts() ([]models.AccountInfo, error) {
	return e.sessionMgr.ListAccounts()
}

// DeleteAccount deletes a specific account from SQLite
func (e *Engine) DeleteAccount(accountID string) error {
	return e.sessionMgr.DeleteAccount(accountID)
}

// LoginInteractive opens a visible browser window, guides the user to log in, and extracts auth_token and ct0 automatically
func (e *Engine) LoginInteractive(timeoutSec int) error {
	if timeoutSec <= 0 {
		timeoutSec = 180
	}

	if err := e.page.Navigate("https://x.com/login"); err != nil {
		return fmt.Errorf("failed to navigate to login page: %w", err)
	}

	fmt.Println("[x-spider-ai] Browser opened. Please log in to your Twitter/X account...")
	fmt.Printf("[x-spider-ai] Waiting for login completion (timeout: %d seconds)...\n", timeoutSec)

	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)

		cookies, err := proto.NetworkGetCookies{Urls: []string{"https://x.com"}}.Call(e.page)
		if err != nil {
			continue
		}

		var authToken, ct0 string
		cookieMap := make(map[string]string)
		for _, c := range cookies.Cookies {
			cookieMap[c.Name] = c.Value
			if c.Name == "auth_token" {
				authToken = c.Value
			}
			if c.Name == "ct0" {
				ct0 = c.Value
			}
		}

		if authToken != "" {
			fmt.Println("[x-spider-ai] Login detected! Successfully captured auth_token and ct0.")
			e.sessionData = &SessionData{
				AuthToken: authToken,
				CT0:       ct0,
				CookieMap: cookieMap,
			}
			if err := e.sessionMgr.SaveSession(e.sessionData); err != nil {
				return fmt.Errorf("failed to save encrypted session: %w", err)
			}
			fmt.Println("[x-spider-ai] Session encrypted and saved successfully.")
			return nil
		}
	}

	return fmt.Errorf("timed out waiting for login")
}

// ClearCapturedTweets resets accumulated tweets in memory
func (e *Engine) ClearCapturedTweets() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.capturedTweets = nil
}

// ClearCapturedUser resets accumulated user profile in memory
func (e *Engine) ClearCapturedUser() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.capturedUser = nil
}

// GetCapturedUser returns the last captured user profile
func (e *Engine) GetCapturedUser() *models.UserProfile {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.capturedUser
}

// GetAndResetLastCreatedTweetID retrieves the captured Tweet ID and resets it
func (e *Engine) GetAndResetLastCreatedTweetID() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	id := e.lastCreatedTweetID
	e.lastCreatedTweetID = ""
	return id
}

// GetCapturedTweets returns a snapshot of intercepted tweets
func (e *Engine) GetCapturedTweets() []models.Tweet {
	e.mu.RLock()
	defer e.mu.RUnlock()
	copied := make([]models.Tweet, len(e.capturedTweets))
	copy(copied, e.capturedTweets)
	return copied
}

// Scroll simulates natural user scrolling to load dynamic content
func (e *Engine) Scroll(opts models.ScrollOptions) ([]models.Tweet, error) {
	if e.page == nil {
		return nil, fmt.Errorf("page not initialized")
	}

	if opts.MaxScrolls <= 0 {
		opts.MaxScrolls = 3
	}
	if opts.DelayMs <= 0 {
		opts.DelayMs = 1500
	}

	initialLen := len(e.GetCapturedTweets())

	for i := 0; i < opts.MaxScrolls; i++ {
		// Evaluate scroll in window or specific element
		if opts.ScrollElement != "" {
			_, _ = e.page.Eval(fmt.Sprintf(`() => {
				const el = document.querySelector(%q);
				if (el) el.scrollBy(0, 800);
			}`, opts.ScrollElement))
		} else {
			_, _ = e.page.Eval(`() => window.scrollBy(0, 800)`)
		}

		time.Sleep(time.Duration(opts.DelayMs) * time.Millisecond)

		if opts.TargetCount > 0 && len(e.GetCapturedTweets())-initialLen >= opts.TargetCount {
			break
		}
	}

	return e.GetCapturedTweets(), nil
}

// NavigateTo loads a target URL with realistic delay
func (e *Engine) NavigateTo(url string) error {
	if e.page == nil {
		if err := e.InitPage(); err != nil {
			return err
		}
	}
	if err := e.page.Navigate(url); err != nil {
		return err
	}

	// Never hang indefinitely waiting for full SPA window load
	_ = e.page.Timeout(10 * time.Second).WaitLoad()
	time.Sleep(2 * time.Second)
	return nil
}

// GetPage returns the underlying rod page for DOM actions
func (e *Engine) GetPage() *rod.Page {
	return e.page
}

// ClearSession deletes stored session credentials in SQLite
func (e *Engine) ClearSession() error {
	e.sessionData = nil
	return e.sessionMgr.ClearSession()
}

// DeleteSessionDB removes the SQLite database file entirely
func (e *Engine) DeleteSessionDB() error {
	e.sessionData = nil
	return e.sessionMgr.DeleteDatabase()
}

// GetDBPath returns SQLite database file path
func (e *Engine) GetDBPath() string {
	return e.sessionMgr.GetDBPath()
}

// Close terminates page router and browser
func (e *Engine) Close() error {
	if e.router != nil {
		_ = e.router.Stop()
	}
	if e.browser != nil {
		return e.browser.Close()
	}
	return nil
}
