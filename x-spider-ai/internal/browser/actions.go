package browser

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"x-spider-ai/internal/models"
)

// Actions provides DOM-level actions (post, reply, quote, like, retweet, bookmark, follow, DM)
type Actions struct {
	engine *Engine
}

func NewActions(eng *Engine) *Actions {
	return &Actions{engine: eng}
}

func (a *Actions) page() *rod.Page {
	return a.engine.GetPage()
}

// PostTweet creates a new Tweet or Reply or Quote Tweet via DOM automation
func (a *Actions) PostTweet(opts models.PostTweetOptions) (*models.ActionResult, error) {
	p := a.page()
	if p == nil {
		return nil, fmt.Errorf("browser page not initialized")
	}

	// If it's a direct reply or quote tweet, handle context
	if opts.InReplyToID != "" {
		replyURL := fmt.Sprintf("https://x.com/i/status/%s", opts.InReplyToID)
		if err := a.engine.NavigateTo(replyURL); err != nil {
			return nil, err
		}
	} else if opts.QuoteTweetID != "" {
		quoteURL := fmt.Sprintf("https://x.com/intent/tweet?quoted_tweet_id=%s", opts.QuoteTweetID)
		if err := a.engine.NavigateTo(quoteURL); err != nil {
			return nil, err
		}
	} else {
		// New compose tweet
		if err := a.engine.NavigateTo("https://x.com/compose/post"); err != nil {
			return nil, err
		}
	}

	time.Sleep(1500 * time.Millisecond)

	// Find the compose editor area
	editor, err := p.Element(`div[data-testid="tweetTextarea_0"]`)
	if err != nil {
		// Fallback editor selector
		editor, err = p.Element(`div[role="textbox"]`)
		if err != nil {
			return nil, fmt.Errorf("tweet compose textbox not found: %w", err)
		}
	}

	// Focus and type text with human typing jitter
	if err := editor.Click(proto.InputMouseButtonLeft, 1); err == nil {
		time.Sleep(300 * time.Millisecond)
		for _, ch := range opts.Text {
			_ = editor.Input(string(ch))
			time.Sleep(20 * time.Millisecond)
		}
	}

	// Handle media attachments if provided
	if len(opts.MediaFilePaths) > 0 {
		fileInput, err := p.Element(`input[data-testid="fileInput"]`)
		if err == nil {
			_ = fileInput.SetFiles(opts.MediaFilePaths)
			time.Sleep(2500 * time.Millisecond) // Wait for upload
		}
	}

	// Click Tweet/Post button
	postBtn, err := p.Element(`button[data-testid="tweetButton"]`)
	if err != nil {
		postBtn, err = p.Element(`button[data-testid="tweetButtonInline"]`)
	}
	if err != nil {
		return nil, fmt.Errorf("could not find post button: %w", err)
	}

	if err := postBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, fmt.Errorf("failed to click post button: %w", err)
	}

	time.Sleep(3000 * time.Millisecond)

	tweetID := a.engine.GetAndResetLastCreatedTweetID()
	tweetURL := ""
	if tweetID != "" {
		tweetURL = fmt.Sprintf("https://x.com/i/status/%s", tweetID)
	}

	return &models.ActionResult{
		Success:  true,
		Action:   "post_tweet",
		TweetID:  tweetID,
		TweetURL: tweetURL,
		Target:   tweetID,
		Data:     opts.Text,
	}, nil
}

// LikeTweet likes a tweet given its ID
func (a *Actions) LikeTweet(tweetID string) (*models.ActionResult, error) {
	return a.toggleTweetAction(tweetID, "like", `button[data-testid="like"]`)
}

// UnlikeTweet un-likes a tweet given its ID
func (a *Actions) UnlikeTweet(tweetID string) (*models.ActionResult, error) {
	return a.toggleTweetAction(tweetID, "unlike", `button[data-testid="unlike"]`)
}

// Retweet reposts a tweet given its ID
func (a *Actions) Retweet(tweetID string) (*models.ActionResult, error) {
	p := a.page()
	tweetURL := fmt.Sprintf("https://x.com/i/status/%s", tweetID)
	if err := a.engine.NavigateTo(tweetURL); err != nil {
		return nil, err
	}

	btn, err := p.Element(`button[data-testid="retweet"]`)
	if err != nil {
		return nil, fmt.Errorf("retweet button not found (may already be retweeted): %w", err)
	}
	_ = btn.Click(proto.InputMouseButtonLeft, 1)
	time.Sleep(500 * time.Millisecond)

	// Confirm retweet popup
	confirmBtn, err := p.Element(`div[data-testid="retweetConfirm"]`)
	if err == nil {
		_ = confirmBtn.Click(proto.InputMouseButtonLeft, 1)
	}

	time.Sleep(1 * time.Second)
	return &models.ActionResult{
		Success: true,
		Action:  "retweet",
		Target:  tweetID,
	}, nil
}

// Unretweet removes a repost
func (a *Actions) Unretweet(tweetID string) (*models.ActionResult, error) {
	p := a.page()
	tweetURL := fmt.Sprintf("https://x.com/i/status/%s", tweetID)
	if err := a.engine.NavigateTo(tweetURL); err != nil {
		return nil, err
	}

	btn, err := p.Element(`button[data-testid="unretweet"]`)
	if err != nil {
		return nil, fmt.Errorf("unretweet button not found: %w", err)
	}
	_ = btn.Click(proto.InputMouseButtonLeft, 1)
	time.Sleep(500 * time.Millisecond)

	confirmBtn, err := p.Element(`div[data-testid="unretweetConfirm"]`)
	if err == nil {
		_ = confirmBtn.Click(proto.InputMouseButtonLeft, 1)
	}

	time.Sleep(1 * time.Second)
	return &models.ActionResult{
		Success: true,
		Action:  "unretweet",
		Target:  tweetID,
	}, nil
}

// BookmarkTweet bookmarks a tweet
func (a *Actions) BookmarkTweet(tweetID string) (*models.ActionResult, error) {
	return a.toggleTweetAction(tweetID, "bookmark", `button[data-testid="bookmark"]`)
}

// UnbookmarkTweet removes a bookmark
func (a *Actions) UnbookmarkTweet(tweetID string) (*models.ActionResult, error) {
	return a.toggleTweetAction(tweetID, "remove_bookmark", `button[data-testid="removeBookmark"]`)
}

// DeleteTweet deletes the authenticated user's tweet
func (a *Actions) DeleteTweet(tweetID string) (*models.ActionResult, error) {
	p := a.page()
	tweetURL := fmt.Sprintf("https://x.com/i/status/%s", tweetID)
	if err := a.engine.NavigateTo(tweetURL); err != nil {
		return nil, err
	}

	// Click tweet caret menu (3 dots)
	caret, err := p.Element(`button[data-testid="caret"]`)
	if err != nil {
		return nil, fmt.Errorf("caret menu not found: %w", err)
	}
	_ = caret.Click(proto.InputMouseButtonLeft, 1)
	time.Sleep(500 * time.Millisecond)

	// Click delete item in menu dropdown
	delItem, err := p.ElementR(`div[role="menuitem"]`, "Delete")
	if err != nil {
		return nil, fmt.Errorf("delete option not found in menu: %w", err)
	}
	_ = delItem.Click(proto.InputMouseButtonLeft, 1)
	time.Sleep(500 * time.Millisecond)

	// Click confirmation button in dialog
	confirmBtn, err := p.Element(`button[data-testid="confirmationSheetConfirm"]`)
	if err != nil {
		return nil, fmt.Errorf("delete confirmation button not found: %w", err)
	}
	_ = confirmBtn.Click(proto.InputMouseButtonLeft, 1)

	time.Sleep(1500 * time.Millisecond)
	return &models.ActionResult{
		Success: true,
		Action:  "delete_tweet",
		Target:  tweetID,
	}, nil
}

// FollowUser follows a target user by their @handle
func (a *Actions) FollowUser(screenName string) (*models.ActionResult, error) {
	screenName = strings.TrimPrefix(screenName, "@")
	p := a.page()
	userURL := fmt.Sprintf("https://x.com/%s", screenName)
	if err := a.engine.NavigateTo(userURL); err != nil {
		return nil, err
	}

	followBtn, err := p.Element(`button[data-testid$="-follow"]`)
	if err != nil {
		return nil, fmt.Errorf("follow button not found (might already be following): %w", err)
	}

	if err := followBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, err
	}

	time.Sleep(1 * time.Second)
	return &models.ActionResult{
		Success: true,
		Action:  "follow_user",
		Target:  screenName,
	}, nil
}

// UnfollowUser unfollows a target user
func (a *Actions) UnfollowUser(screenName string) (*models.ActionResult, error) {
	screenName = strings.TrimPrefix(screenName, "@")
	p := a.page()
	userURL := fmt.Sprintf("https://x.com/%s", screenName)
	if err := a.engine.NavigateTo(userURL); err != nil {
		return nil, err
	}

	unfollowBtn, err := p.Element(`button[data-testid$="-unfollow"]`)
	if err != nil {
		return nil, fmt.Errorf("unfollow button not found: %w", err)
	}

	if err := unfollowBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, err
	}
	time.Sleep(500 * time.Millisecond)

	confirmBtn, err := p.Element(`button[data-testid="confirmationSheetConfirm"]`)
	if err == nil {
		_ = confirmBtn.Click(proto.InputMouseButtonLeft, 1)
	}

	time.Sleep(1 * time.Second)
	return &models.ActionResult{
		Success: true,
		Action:  "unfollow_user",
		Target:  screenName,
	}, nil
}

// SendDirectMessage sends a direct message to a user
func (a *Actions) SendDirectMessage(screenName, text string) (*models.ActionResult, error) {
	screenName = strings.TrimPrefix(screenName, "@")
	p := a.page()
	dmURL := fmt.Sprintf("https://x.com/messages/compose?recipient_id=%s", screenName)
	if err := a.engine.NavigateTo(dmURL); err != nil {
		return nil, err
	}

	time.Sleep(1500 * time.Millisecond)
	inputBox, err := p.Element(`div[data-testid="dmComposerTextInput"]`)
	if err != nil {
		inputBox, err = p.Element(`aside div[role="textbox"]`)
	}
	if err != nil {
		return nil, fmt.Errorf("dm composer textbox not found: %w", err)
	}

	_ = inputBox.Click(proto.InputMouseButtonLeft, 1)
	time.Sleep(200 * time.Millisecond)
	for _, ch := range text {
		_ = inputBox.Input(string(ch))
		time.Sleep(15 * time.Millisecond)
	}

	sendBtn, err := p.Element(`button[data-testid="dmComposerSendButton"]`)
	if err != nil {
		return nil, fmt.Errorf("dm send button not found: %w", err)
	}
	_ = sendBtn.Click(proto.InputMouseButtonLeft, 1)
	time.Sleep(1500 * time.Millisecond)

	return &models.ActionResult{
		Success: true,
		Action:  "send_dm",
		Target:  screenName,
		Data:    text,
	}, nil
}

func (a *Actions) toggleTweetAction(tweetID, actionName, selector string) (*models.ActionResult, error) {
	p := a.page()
	tweetURL := fmt.Sprintf("https://x.com/i/status/%s", tweetID)
	if err := a.engine.NavigateTo(tweetURL); err != nil {
		return nil, err
	}

	btn, err := p.Element(selector)
	if err != nil {
		return nil, fmt.Errorf("action button %q not found for tweet %s: %w", selector, tweetID, err)
	}

	if err := btn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, fmt.Errorf("failed to click action button: %w", err)
	}

	time.Sleep(1 * time.Second)
	return &models.ActionResult{
		Success: true,
		Action:  actionName,
		Target:  tweetID,
	}, nil
}
