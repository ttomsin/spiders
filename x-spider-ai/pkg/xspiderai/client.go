package xspiderai

import (
	"fmt"
	"x-spider-ai/internal/browser"
)

// Client is the primary SDK interface for AI agents and applications to interact with Twitter/X
type Client struct {
	engine  *browser.Engine
	actions *browser.Actions
	reader  *browser.Reader
}

// ClientOptions allows configuring the underlying engine
type ClientOptions struct {
	Headless    bool
	ProxyURL    string
	SessionPath string
	UserDataDir string
}

// NewClient creates an initialized x-spider-ai client
func NewClient(opts ClientOptions) (*Client, error) {
	eng, err := browser.NewEngine(browser.Config{
		Headless:    opts.Headless,
		ProxyURL:    opts.ProxyURL,
		SessionPath: opts.SessionPath,
		UserDataDir: opts.UserDataDir,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize x-spider-ai engine: %w", err)
	}

	if err := eng.InitPage(); err != nil {
		_ = eng.Close()
		return nil, fmt.Errorf("failed to initialize page: %w", err)
	}

	return &Client{
		engine:  eng,
		actions: browser.NewActions(eng),
		reader:  browser.NewReader(eng),
	}, nil
}

// Login sets Twitter authentication tokens (auth_token and optional ct0)
func (c *Client) Login(authToken, ct0 string) error {
	return c.engine.LoginWithTokens(authToken, ct0)
}

// LoginInteractive opens the browser for the user to log in manually, then captures tokens automatically
func (c *Client) LoginInteractive(timeoutSec int) error {
	return c.engine.LoginInteractive(timeoutSec)
}

// PostTweet posts a new tweet, reply, or quote tweet (with optional media)
func (c *Client) PostTweet(opts PostTweetOptions) (*ActionResult, error) {
	return c.actions.PostTweet(opts)
}

// ReplyTweet is a convenience method to reply to a specific tweet ID
func (c *Client) ReplyTweet(tweetID, text string, mediaPaths ...string) (*ActionResult, error) {
	return c.actions.PostTweet(PostTweetOptions{
		Text:           text,
		InReplyToID:    tweetID,
		MediaFilePaths: mediaPaths,
	})
}

// QuoteTweet is a convenience method to quote a specific tweet ID
func (c *Client) QuoteTweet(tweetID, text string, mediaPaths ...string) (*ActionResult, error) {
	return c.actions.PostTweet(PostTweetOptions{
		Text:           text,
		QuoteTweetID:   tweetID,
		MediaFilePaths: mediaPaths,
	})
}

// LikeTweet likes a tweet
func (c *Client) LikeTweet(tweetID string) (*ActionResult, error) {
	return c.actions.LikeTweet(tweetID)
}

// UnlikeTweet un-likes a tweet
func (c *Client) UnlikeTweet(tweetID string) (*ActionResult, error) {
	return c.actions.UnlikeTweet(tweetID)
}

// Retweet reposts a tweet
func (c *Client) Retweet(tweetID string) (*ActionResult, error) {
	return c.actions.Retweet(tweetID)
}

// Unretweet removes a repost
func (c *Client) Unretweet(tweetID string) (*ActionResult, error) {
	return c.actions.Unretweet(tweetID)
}

// BookmarkTweet bookmarks a tweet
func (c *Client) BookmarkTweet(tweetID string) (*ActionResult, error) {
	return c.actions.BookmarkTweet(tweetID)
}

// UnbookmarkTweet removes a bookmark
func (c *Client) UnbookmarkTweet(tweetID string) (*ActionResult, error) {
	return c.actions.UnbookmarkTweet(tweetID)
}

// DeleteTweet deletes an authenticated user's tweet
func (c *Client) DeleteTweet(tweetID string) (*ActionResult, error) {
	return c.actions.DeleteTweet(tweetID)
}

// FollowUser follows a user by handle
func (c *Client) FollowUser(screenName string) (*ActionResult, error) {
	return c.actions.FollowUser(screenName)
}

// UnfollowUser unfollows a user by handle
func (c *Client) UnfollowUser(screenName string) (*ActionResult, error) {
	return c.actions.UnfollowUser(screenName)
}

// SendDirectMessage sends a direct message to a user
func (c *Client) SendDirectMessage(screenName, text string) (*ActionResult, error) {
	return c.actions.SendDirectMessage(screenName, text)
}

// ReadThread reads a tweet and scrolls through its reply thread
func (c *Client) ReadThread(tweetID string, scrollOpts ScrollOptions) (*ThreadDetails, error) {
	return c.reader.ReadThread(tweetID, scrollOpts)
}

// SearchTweets searches Twitter and scrolls through results
func (c *Client) SearchTweets(query string, tab string, scrollOpts ScrollOptions) ([]Tweet, error) {
	return c.reader.SearchTweets(query, tab, scrollOpts)
}

// ReadUserTimeline reads tweets from a specific user's profile
func (c *Client) ReadUserTimeline(screenName string, scrollOpts ScrollOptions) ([]Tweet, error) {
	return c.reader.ReadUserTimeline(screenName, scrollOpts)
}

// GetProfile fetches user profile details by handle
func (c *Client) GetProfile(screenName string) (*UserProfile, error) {
	return c.reader.GetProfile(screenName)
}

// GetMyProfile fetches the authenticated account's profile details
func (c *Client) GetMyProfile() (*UserProfile, error) {
	return c.reader.GetMyProfile()
}

// Scroll manually scrolls the current page to trigger network interception of new items
func (c *Client) Scroll(opts ScrollOptions) ([]Tweet, error) {
	return c.engine.Scroll(opts)
}

// ClearSession deletes saved credentials in SQLite
func (c *Client) ClearSession() error {
	return c.engine.ClearSession()
}

// DeleteDatabase completely removes the SQLite sessions database file
func (c *Client) DeleteDatabase() error {
	return c.engine.DeleteSessionDB()
}

// GetDBPath returns the filesystem path of the SQLite database
func (c *Client) GetDBPath() string {
	return c.engine.GetDBPath()
}

// Close gracefully closes the underlying browser
func (c *Client) Close() error {
	return c.engine.Close()
}
