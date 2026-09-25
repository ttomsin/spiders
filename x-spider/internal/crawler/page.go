package crawler

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"x-spider/internal/config"
)

const (
	baseRateLimitTimeout    = 60 * time.Second
	maximumRateLimitTimeout = 600 * time.Second
	rateLimitRatio          = 2
)

// CalculateForRateLimit calculates backoff delay when hitting Twitter rate limit
func CalculateForRateLimit(attempt int, enabled bool) time.Duration {
	if !enabled {
		return baseRateLimitTimeout
	}
	timeout := time.Duration(rateLimitRatio*attempt)*baseRateLimitTimeout + baseRateLimitTimeout
	if timeout > maximumRateLimitTimeout {
		return maximumRateLimitTimeout
	}
	return timeout
}

// ScrollUp scrolls smoothly to the top of the browser page
func ScrollUp(page *rod.Page) error {
	_, err := page.Eval(`() => {
		window.scrollTo({
			behavior: "smooth",
			top: 0,
		});
	}`)
	return err
}

// ScrollDown scrolls to the bottom and removes heavy media nodes from the DOM
func ScrollDown(page *rod.Page) error {
	_, err := page.Eval(`() => {
		window.scrollTo({
			behavior: "smooth",
			top: document.body.scrollHeight,
		});

		// Remove elements with tweetPhoto testid to conserve browser memory
		document.querySelectorAll("a div[data-testid='tweetPhoto']").forEach(el => el.remove());
		document.querySelectorAll("a div[aria-label='Image']").forEach(el => el.remove());
		document.querySelectorAll("div[data-testid='tweetPhoto']").forEach(el => el.remove());
	}`)
	return err
}

// FormatDateForTwitter formats DD-MM-YYYY to YYYY-MM-DD for Twitter search queries
func FormatDateForTwitter(d string) string {
	parts := strings.Split(strings.TrimSpace(d), "-")
	if len(parts) == 3 {
		// input is DD-MM-YYYY -> output YYYY-MM-DD
		return fmt.Sprintf("%s-%s-%s", parts[2], parts[1], parts[0])
	}
	return d
}

// BuildSearchKeyword constructs the final search query including date filters
func BuildSearchKeyword(cfg *config.Config) string {
	keyword := cfg.SearchKeyword
	if cfg.FromDate != "" {
		formattedFrom := FormatDateForTwitter(cfg.FromDate)
		keyword += fmt.Sprintf(" since:%s", formattedFrom)
	}
	if cfg.ToDate != "" {
		formattedTo := FormatDateForTwitter(cfg.ToDate)
		keyword += fmt.Sprintf(" until:%s", formattedTo)
	}
	return keyword
}

// InputKeywords fills in the Twitter advanced search inputs and presses Enter
func InputKeywords(page *rod.Page, cfg *config.Config) error {
	selector := `input[name="allOfTheseWords"]`
	el, err := page.Timeout(30 * time.Second).Element(selector)
	if err != nil {
		return fmt.Errorf("search input element not found (%s): %w", selector, err)
	}

	if err := el.WaitVisible(); err != nil {
		return fmt.Errorf("search input element not visible: %w", err)
	}

	if err := el.Click("left", 1); err != nil {
		return fmt.Errorf("failed to click search input: %w", err)
	}

	query := BuildSearchKeyword(cfg)
	yellow := color.New(color.FgYellow).SprintfFunc()
	fmt.Printf("\n%s\n\n", yellow("Filling in keywords: %s", query))

	// Clear and input keywords
	if err := el.SelectAllText(); err == nil {
		_ = el.Input(query)
	} else {
		if err := el.Input(query); err != nil {
			return fmt.Errorf("failed to type keywords: %w", err)
		}
	}

	// Press Enter to trigger search
	return el.Type(input.Enter)
}
