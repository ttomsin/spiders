package browser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"chatgpt-spider/internal/auth"
)

// Options holds browser launch options
type Options struct {
	Headless     bool
	Anonymous    bool // Do not load saved credentials or user profile
	UserDataDir  string
	SessionToken string
	Debug        bool
}

// BrowserInstance manages the Rod browser and page lifecycle
type BrowserInstance struct {
	Browser     *rod.Browser
	Page        *rod.Page
	tempDataDir string
}

// Launch launches a stealth browser connected to ChatGPT
func Launch(opts Options) (*BrowserInstance, error) {
	userDataDir := opts.UserDataDir
	var tempDataDir string

	if opts.Anonymous {
		// Use ephemeral temp directory for clean unauthenticated session
		tDir, err := os.MkdirTemp("", "chatgpt_anon_*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp dir: %w", err)
		}
		userDataDir = tDir
		tempDataDir = tDir
	} else if userDataDir == "" {
		dir, err := auth.GetProfileDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get profile directory: %w", err)
		}
		userDataDir = dir
	}

	_ = os.MkdirAll(userDataDir, 0700)

	// Remove stale lockfiles if leftover from a previous crash/termination
	_ = os.Remove(filepath.Join(userDataDir, "lockfile"))
	_ = os.Remove(filepath.Join(userDataDir, "SingletonLock"))
	_ = os.Remove(filepath.Join(userDataDir, "SingletonSocket"))
	_ = os.Remove(filepath.Join(userDataDir, "SingletonCookie"))

	l := launcher.New().
		Leakless(false).
		Headless(opts.Headless).
		UserDataDir(userDataDir).
		Devtools(opts.Debug).
		Set("disable-background-networking").
		Set("disable-background-timer-throttling").
		Set("disable-backgrounding-occluded-windows").
		Set("disable-breakpad").
		Set("disable-client-side-phishing-detection").
		Set("disable-default-apps").
		Set("disable-dev-shm-usage").
		Set("disable-extensions").
		Set("disable-features", "TranslateUI").
		Set("disable-hang-monitor").
		Set("disable-ipc-flooding-protection").
		Set("disable-popup-blocking").
		Set("disable-prompt-on-repost").
		Set("disable-renderer-backgrounding").
		Set("disable-sync").
		Set("force-color-profile", "srgb").
		Set("metrics-recording-only").
		Set("no-first-run")

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

	// Inject session cookies if stored or provided (only in authenticated mode)
	if !opts.Anonymous {
		cookies, _ := auth.GetSessionCookies()
		if cookies == nil {
			cookies = make(map[string]string)
		}

		if opts.SessionToken != "" {
			if strings.Contains(opts.SessionToken, "=") {
				// Format: "name=value" or multiple separated by semicolon
				parts := strings.Split(opts.SessionToken, ";")
				for _, p := range parts {
					kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
					if len(kv) == 2 {
						cookies[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
					}
				}
			} else {
				cookies["__Secure-next-auth.session-token"] = opts.SessionToken
			}
		}

		if len(cookies) > 0 {
			var cookieParams []*proto.NetworkCookieParam
			for name, val := range cookies {
				for _, domain := range []string{".chatgpt.com", ".openai.com"} {
					cookieParams = append(cookieParams, &proto.NetworkCookieParam{
						Name:     name,
						Value:    val,
						Domain:   domain,
						Path:     "/",
						HTTPOnly: true,
						Secure:   true,
						SameSite: proto.NetworkCookieSameSiteLax,
					})
				}
			}
			_ = page.SetCookies(cookieParams)
		}
	}

	return &BrowserInstance{
		Browser:     b,
		Page:        page,
		tempDataDir: tempDataDir,
	}, nil
}

// Close gracefully closes the browser instance and cleans any ephemeral data
func (bi *BrowserInstance) Close() error {
	var err error
	if bi.Browser != nil {
		err = bi.Browser.Close()
	}
	if bi.tempDataDir != "" {
		_ = os.RemoveAll(bi.tempDataDir)
	}
	return err
}