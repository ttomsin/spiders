package browser

import (
	"fmt"
	"os"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"chatgpt-spider/internal/auth"
)

// Options holds browser launch options
type Options struct {
	Headless    bool
	UserDataDir string
	SessionToken string
	Debug       bool
}

// BrowserInstance manages the Rod browser and page lifecycle
type BrowserInstance struct {
	Browser *rod.Browser
	Page    *rod.Page
}

// Launch launches a stealth browser connected to ChatGPT
func Launch(opts Options) (*BrowserInstance, error) {
	userDataDir := opts.UserDataDir
	if userDataDir == "" {
		dir, err := auth.GetProfileDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get profile directory: %w", err)
		}
		userDataDir = dir
	}

	_ = os.MkdirAll(userDataDir, 0700)

	l := launcher.New().
		Leakless(false).
		Headless(opts.Headless).
		UserDataDir(userDataDir).
		Devtools(opts.Debug)

	u, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch chromium: %w", err)
	}

	b := rod.New().ControlURL(u)
	if err := b.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}

	page, err := stealth.Page(b)
	if err != nil {
		_ = b.Close()
		return nil, fmt.Errorf("failed to create stealth page: %w", err)
	}

	_ = page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:             1280,
		Height:            900,
		DeviceScaleFactor: 1,
	})

	// Inject session token cookie if provided
	token := opts.SessionToken
	if token == "" {
		if stored, err := auth.GetSessionToken(); err == nil && stored != "" {
			token = stored
		}
	}

	if token != "" {
		_ = page.SetCookies([]*proto.NetworkCookieParam{
			{
				Name:     "__Secure-next-auth.session-token",
				Value:    token,
				Domain:   ".chatgpt.com",
				Path:     "/",
				HTTPOnly: true,
				Secure:   true,
				SameSite: proto.NetworkCookieSameSiteLax,
			},
			{
				Name:     "__Secure-next-auth.session-token",
				Value:    token,
				Domain:   ".openai.com",
				Path:     "/",
				HTTPOnly: true,
				Secure:   true,
				SameSite: proto.NetworkCookieSameSiteLax,
			},
		})
	}

	return &BrowserInstance{
		Browser: b,
		Page:    page,
	}, nil
}

// Close gracefully closes the browser instance
func (bi *BrowserInstance) Close() error {
	if bi.Browser != nil {
		return bi.Browser.Close()
	}
	return nil
}