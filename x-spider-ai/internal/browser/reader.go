package browser

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"x-spider-ai/internal/models"
)

// Reader provides high-level timeline, search, thread, and reply reading capabilities
type Reader struct {
	engine *Engine
}

func NewReader(eng *Engine) *Reader {
	return &Reader{engine: eng}
}

// ReadThread navigates to a specific tweet, scrolls through replies, and returns the main tweet + replies
func (r *Reader) ReadThread(tweetID string, scrollOpts models.ScrollOptions) (*models.ThreadDetails, error) {
	tweetURL := fmt.Sprintf("https://x.com/i/status/%s", tweetID)

	r.engine.ClearCapturedTweets()

	if err := r.engine.NavigateTo(tweetURL); err != nil {
		return nil, fmt.Errorf("failed to navigate to tweet thread: %w", err)
	}

	// Scroll down to load replies
	if scrollOpts.MaxScrolls <= 0 {
		scrollOpts.MaxScrolls = 4
	}
	captured, err := r.engine.Scroll(scrollOpts)
	if err != nil {
		return nil, err
	}

	details := &models.ThreadDetails{
		Replies: make([]models.Tweet, 0),
	}

	seen := make(map[string]bool)
	for _, tw := range captured {
		if seen[tw.ID] {
			continue
		}
		seen[tw.ID] = true

		if tw.ID == tweetID {
			details.MainTweet = tw
		} else {
			details.Replies = append(details.Replies, tw)
		}
	}

	return details, nil
}

// SearchTweets performs a search on X and captures tweets from the timeline
func (r *Reader) SearchTweets(query string, tab string, scrollOpts models.ScrollOptions) ([]models.Tweet, error) {
	// tab: "Top" (default) or "Latest"
	fParam := ""
	if strings.EqualFold(tab, "latest") {
		fParam = "&f=live"
	}

	searchURL := fmt.Sprintf("https://x.com/search?q=%s%s", url.QueryEscape(query), fParam)
	r.engine.ClearCapturedTweets()

	if err := r.engine.NavigateTo(searchURL); err != nil {
		return nil, fmt.Errorf("failed to navigate to search timeline: %w", err)
	}

	if scrollOpts.MaxScrolls <= 0 {
		scrollOpts.MaxScrolls = 3
	}

	captured, err := r.engine.Scroll(scrollOpts)
	if err != nil {
		return nil, err
	}

	// Deduplicate
	var results []models.Tweet
	seen := make(map[string]bool)
	for _, tw := range captured {
		if !seen[tw.ID] {
			seen[tw.ID] = true
			results = append(results, tw)
		}
	}

	return results, nil
}

// Discover executes multi-query searches with temporal and engagement constraints
func (r *Reader) Discover(opts models.DiscoverOptions) ([]models.Tweet, error) {
	if len(opts.Queries) == 0 {
		return nil, fmt.Errorf("at least one search query must be provided")
	}

	maxResults := opts.MaxResults
	if maxResults <= 0 {
		maxResults = 20
	}

	sortMode := strings.ToLower(strings.TrimSpace(opts.Sort))
	if sortMode == "" {
		sortMode = "relevance"
	}

	var combined []models.Tweet
	seen := make(map[string]bool)

	for _, rawQuery := range opts.Queries {
		q := strings.TrimSpace(rawQuery)
		if q == "" {
			continue
		}

		// Build Twitter advanced search clauses
		var queryParts []string
		queryParts = append(queryParts, q)

		if opts.Since != "" {
			queryParts = append(queryParts, fmt.Sprintf("since:%s", opts.Since))
		}
		if opts.Until != "" {
			queryParts = append(queryParts, fmt.Sprintf("until:%s", opts.Until))
		}
		if opts.MinEngagement > 0 {
			queryParts = append(queryParts, fmt.Sprintf("min_faves:%d", opts.MinEngagement))
		}
		if !opts.IncludeReplies {
			queryParts = append(queryParts, "-filter:replies")
		}

		fullQuery := strings.Join(queryParts, " ")

		fParam := ""
		if sortMode == "recent" || sortMode == "oldest" {
			fParam = "&f=live"
		}

		searchURL := fmt.Sprintf("https://x.com/search?q=%s%s", url.QueryEscape(fullQuery), fParam)
		r.engine.ClearCapturedTweets()

		if err := r.engine.NavigateTo(searchURL); err != nil {
			continue
		}

		scrollOpts := models.ScrollOptions{
			MaxScrolls:  2,
			DelayMs:     1200,
			TargetCount: maxResults,
		}

		captured, err := r.engine.Scroll(scrollOpts)
		if err == nil {
			for _, tw := range captured {
				if tw.ID != "" && !seen[tw.ID] {
					// Client-side filtering check for replies if needed
					if !opts.IncludeReplies && tw.InReplyToStatusID != "" {
						continue
					}
					// Client-side check for min engagement
					totalEngagement := tw.FavoriteCount + tw.RetweetCount
					if opts.MinEngagement > 0 && totalEngagement < opts.MinEngagement {
						continue
					}

					seen[tw.ID] = true
					combined = append(combined, tw)
				}
			}
		}

		if len(combined) >= maxResults*2 {
			break
		}
	}

	// Apply post-processing sorts
	switch sortMode {
	case "engagement":
		sort.SliceStable(combined, func(i, j int) bool {
			scoreI := combined[i].FavoriteCount*2 + combined[i].RetweetCount*3 + combined[i].ReplyCount
			scoreJ := combined[j].FavoriteCount*2 + combined[j].RetweetCount*3 + combined[j].ReplyCount
			return scoreI > scoreJ
		})
	case "oldest":
		sort.SliceStable(combined, func(i, j int) bool {
			tI, errI := time.Parse(time.RubyDate, combined[i].CreatedAt)
			tJ, errJ := time.Parse(time.RubyDate, combined[j].CreatedAt)
			if errI == nil && errJ == nil {
				return tI.Before(tJ)
			}
			return combined[i].ID < combined[j].ID
		})
	case "recent":
		sort.SliceStable(combined, func(i, j int) bool {
			tI, errI := time.Parse(time.RubyDate, combined[i].CreatedAt)
			tJ, errJ := time.Parse(time.RubyDate, combined[j].CreatedAt)
			if errI == nil && errJ == nil {
				return tI.After(tJ)
			}
			return combined[i].ID > combined[j].ID
		})
	case "relevance":
		// Preserves Twitter's original search relevance ranking
	}

	if len(combined) > maxResults {
		combined = combined[:maxResults]
	}

	return combined, nil
}

// ReadUserTimeline navigates to a user's profile and returns their recent tweets
func (r *Reader) ReadUserTimeline(screenName string, scrollOpts models.ScrollOptions) ([]models.Tweet, error) {
	screenName = strings.TrimPrefix(screenName, "@")
	userURL := fmt.Sprintf("https://x.com/%s", screenName)
	r.engine.ClearCapturedTweets()

	if err := r.engine.NavigateTo(userURL); err != nil {
		return nil, fmt.Errorf("failed to navigate to user profile: %w", err)
	}

	if scrollOpts.MaxScrolls <= 0 {
		scrollOpts.MaxScrolls = 3
	}

	captured, err := r.engine.Scroll(scrollOpts)
	if err != nil {
		return nil, err
	}

	var results []models.Tweet
	seen := make(map[string]bool)
	for _, tw := range captured {
		if !seen[tw.ID] {
			seen[tw.ID] = true
			results = append(results, tw)
		}
	}

	return results, nil
}

// GetProfile fetches details for any user by their screen name
func (r *Reader) GetProfile(screenName string) (*models.UserProfile, error) {
	screenName = strings.TrimPrefix(screenName, "@")
	userURL := fmt.Sprintf("https://x.com/%s", screenName)
	r.engine.ClearCapturedUser()

	if err := r.engine.NavigateTo(userURL); err != nil {
		return nil, fmt.Errorf("failed to navigate to profile: %w", err)
	}

	// Check for intercepted GraphQL user profile payload
	for i := 0; i < 5; i++ {
		time.Sleep(300 * time.Millisecond)
		if u := r.engine.GetCapturedUser(); u != nil && strings.EqualFold(u.ScreenName, screenName) {
			return u, nil
		}
	}

	// Fallback: extract from DOM
	p := r.engine.GetPage()
	if p != nil {
		nameText := screenName
		if nameEl, err := p.Element(`div[data-testid="UserName"]`); err == nil {
			rawName, _ := nameEl.Text()
			lines := strings.Split(rawName, "\n")
			if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
				nameText = strings.TrimSpace(lines[0])
			}
		}

		bioText := ""
		if bioEl, err := p.Element(`div[data-testid="UserDescription"]`); err == nil {
			bioText, _ = bioEl.Text()
		}

		// Follower and Following counts from DOM links or profile header
		followersCount := 0
		followingCount := 0
		countsRes, err := p.Eval(`() => {
			let followers = 0;
			let following = 0;
			const links = document.querySelectorAll('a[href*="/verified_followers"], a[href$="/followers"], a[href$="/following"]');
			for (const a of links) {
				const href = a.getAttribute('href') || '';
				const text = a.innerText || '';
				const numMatch = text.match(/([\d,]+(\.\d+)?[KkMm]?)/);
				if (numMatch) {
					let val = numMatch[1].replace(/,/g, '');
					let mult = 1;
					if (val.endsWith('K') || val.endsWith('k')) {
						mult = 1000;
						val = val.slice(0, -1);
					} else if (val.endsWith('M') || val.endsWith('m')) {
						mult = 1000000;
						val = val.slice(0, -1);
					}
					const parsed = Math.round(parseFloat(val) * mult);
					if (href.includes('following')) {
						following = parsed;
					} else if (href.includes('followers')) {
						followers = parsed;
					}
				}
			}
			return { followers, following };
		}`)
		if err == nil {
			m := countsRes.Value.Map()
			if f, ok := m["followers"]; ok {
				followersCount = int(f.Int())
			}
			if f, ok := m["following"]; ok {
				followingCount = int(f.Int())
			}
		}

		return &models.UserProfile{
			ScreenName:     screenName,
			Name:           nameText,
			Description:    bioText,
			FollowersCount: followersCount,
			FriendsCount:   followingCount,
		}, nil
	}

	return nil, fmt.Errorf("could not extract profile for %s", screenName)
}

func parseCountStr(s string) int {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	var n int
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n
}

// GetMyProfile fetches the authenticated account's profile details
func (r *Reader) GetMyProfile() (*models.UserProfile, error) {
	r.engine.ClearCapturedUser()

	if err := r.engine.NavigateTo("https://x.com/home"); err != nil {
		return nil, err
	}
	time.Sleep(2 * time.Second)

	p := r.engine.GetPage()
	if p == nil {
		return nil, fmt.Errorf("page not available")
	}

	// Use timed element lookups to avoid hanging indefinitely
	pageWithTimeout := p.Timeout(5 * time.Second)

	// Strategy 1: Check side navigation account switcher text
	if btn, err := pageWithTimeout.Element(`button[data-testid="SideNav_AccountSwitcher_Button"]`); err == nil && btn != nil {
		if txt, _ := btn.Text(); txt != "" {
			for _, part := range strings.Fields(txt) {
				if strings.HasPrefix(part, "@") {
					handle := strings.TrimPrefix(part, "@")
					if handle != "" {
						return r.GetProfile(handle)
					}
				}
			}
		}
	}

	// Strategy 2: Check Profile navigation link
	if profBtn, err := pageWithTimeout.Element(`a[data-testid="AppTabBar_Profile_Link"]`); err == nil && profBtn != nil {
		if href, _ := profBtn.Attribute("href"); href != nil && *href != "" {
			handle := strings.TrimPrefix(strings.TrimSpace(*href), "/")
			if handle != "" {
				return r.GetProfile(handle)
			}
		}
	}

	// Strategy 3: Check any anchor matching user profile pattern in left rail
	res, err := p.Eval(`() => {
		const link = document.querySelector('a[data-testid="AppTabBar_Profile_Link"]');
		if (link && link.getAttribute('href')) return link.getAttribute('href');
		const switcher = document.querySelector('button[data-testid="SideNav_AccountSwitcher_Button"]');
		if (switcher && switcher.innerText) {
			const m = switcher.innerText.match(/@([a-zA-Z0-9_]+)/);
			if (m) return m[1];
		}
		return "";
	}`)
	if err == nil && res.Value.Str() != "" {
		handle := strings.TrimPrefix(strings.TrimSpace(res.Value.Str()), "/")
		if handle != "" {
			return r.GetProfile(handle)
		}
	}

	return nil, fmt.Errorf("could not identify current authenticated profile")
}
