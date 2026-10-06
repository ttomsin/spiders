package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"x-spider-ai/pkg/xspiderai"
)

// Server encapsulates the MCP Server exposing x-spider-ai tools to AI agents
type Server struct {
	mcpServer *server.MCPServer
	client    *xspiderai.Client
}

// NewServer initializes an MCP server exposing Twitter automation tools
func NewServer(client *xspiderai.Client) *Server {
	s := server.NewMCPServer(
		"x-spider-ai",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	srv := &Server{
		mcpServer: s,
		client:    client,
	}

	srv.registerTools()
	return srv
}

func (s *Server) registerTools() {
	// Tool: login
	s.mcpServer.AddTool(mcp.NewTool("x_login",
		mcp.WithDescription("Authenticate x-spider-ai with Twitter auth_token and optional ct0 cookie"),
		mcp.WithString("auth_token", mcp.Required(), mcp.Description("Twitter auth_token cookie")),
		mcp.WithString("ct0", mcp.Description("Optional CSRF ct0 cookie")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		authToken, _ := req.GetArguments()["auth_token"].(string)
		ct0, _ := req.GetArguments()["ct0"].(string)

		if err := s.client.Login(authToken, ct0); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Login failed: %v", err)), nil
		}
		return mcp.NewToolResultText("Successfully authenticated and session saved"), nil
	})

	// Tool: post_tweet
	s.mcpServer.AddTool(mcp.NewTool("x_post_tweet",
		mcp.WithDescription("Post a new tweet, optionally replying to another tweet or quoting another tweet"),
		mcp.WithString("text", mcp.Required(), mcp.Description("Text of the tweet to post")),
		mcp.WithString("in_reply_to_id", mcp.Description("Tweet ID if this is a reply to another tweet")),
		mcp.WithString("quote_tweet_id", mcp.Description("Tweet ID if this is quoting another tweet")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		text, _ := req.GetArguments()["text"].(string)
		inReplyTo, _ := req.GetArguments()["in_reply_to_id"].(string)
		quoteTweetID, _ := req.GetArguments()["quote_tweet_id"].(string)

		res, err := s.client.PostTweet(xspiderai.PostTweetOptions{
			Text:         text,
			InReplyToID:  inReplyTo,
			QuoteTweetID: quoteTweetID,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to post tweet: %v", err)), nil
		}

		resBytes, _ := json.Marshal(res)
		return mcp.NewToolResultText(string(resBytes)), nil
	})

	// Tool: like_tweet
	s.mcpServer.AddTool(mcp.NewTool("x_like_tweet",
		mcp.WithDescription("Like a tweet by its tweet ID"),
		mcp.WithString("tweet_id", mcp.Required(), mcp.Description("ID of the tweet to like")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tweetID, _ := req.GetArguments()["tweet_id"].(string)
		res, err := s.client.LikeTweet(tweetID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to like tweet: %v", err)), nil
		}
		resBytes, _ := json.Marshal(res)
		return mcp.NewToolResultText(string(resBytes)), nil
	})

	// Tool: unlike_tweet
	s.mcpServer.AddTool(mcp.NewTool("x_unlike_tweet",
		mcp.WithDescription("Unlike a tweet by its tweet ID"),
		mcp.WithString("tweet_id", mcp.Required(), mcp.Description("ID of the tweet to unlike")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tweetID, _ := req.GetArguments()["tweet_id"].(string)
		res, err := s.client.UnlikeTweet(tweetID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to unlike tweet: %v", err)), nil
		}
		resBytes, _ := json.Marshal(res)
		return mcp.NewToolResultText(string(resBytes)), nil
	})

	// Tool: retweet
	s.mcpServer.AddTool(mcp.NewTool("x_retweet",
		mcp.WithDescription("Retweet / Repost a tweet by ID"),
		mcp.WithString("tweet_id", mcp.Required(), mcp.Description("ID of the tweet to retweet")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tweetID, _ := req.GetArguments()["tweet_id"].(string)
		res, err := s.client.Retweet(tweetID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to retweet: %v", err)), nil
		}
		resBytes, _ := json.Marshal(res)
		return mcp.NewToolResultText(string(resBytes)), nil
	})

	// Tool: bookmark_tweet
	s.mcpServer.AddTool(mcp.NewTool("x_bookmark_tweet",
		mcp.WithDescription("Bookmark a tweet by ID"),
		mcp.WithString("tweet_id", mcp.Required(), mcp.Description("ID of the tweet to bookmark")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tweetID, _ := req.GetArguments()["tweet_id"].(string)
		res, err := s.client.BookmarkTweet(tweetID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to bookmark tweet: %v", err)), nil
		}
		resBytes, _ := json.Marshal(res)
		return mcp.NewToolResultText(string(resBytes)), nil
	})

	// Tool: delete_tweet
	s.mcpServer.AddTool(mcp.NewTool("x_delete_tweet",
		mcp.WithDescription("Delete your own tweet by ID"),
		mcp.WithString("tweet_id", mcp.Required(), mcp.Description("ID of the tweet to delete")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tweetID, _ := req.GetArguments()["tweet_id"].(string)
		res, err := s.client.DeleteTweet(tweetID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to delete tweet: %v", err)), nil
		}
		resBytes, _ := json.Marshal(res)
		return mcp.NewToolResultText(string(resBytes)), nil
	})

	// Tool: follow_user
	s.mcpServer.AddTool(mcp.NewTool("x_follow_user",
		mcp.WithDescription("Follow a user by screen name (@handle)"),
		mcp.WithString("screen_name", mcp.Required(), mcp.Description("Username / handle of the user")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		screenName, _ := req.GetArguments()["screen_name"].(string)
		res, err := s.client.FollowUser(screenName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to follow user: %v", err)), nil
		}
		resBytes, _ := json.Marshal(res)
		return mcp.NewToolResultText(string(resBytes)), nil
	})

	// Tool: send_direct_message
	s.mcpServer.AddTool(mcp.NewTool("x_send_direct_message",
		mcp.WithDescription("Send a direct message to a user"),
		mcp.WithString("screen_name", mcp.Required(), mcp.Description("Username / handle of the recipient")),
		mcp.WithString("text", mcp.Required(), mcp.Description("Message text content")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		screenName, _ := req.GetArguments()["screen_name"].(string)
		text, _ := req.GetArguments()["text"].(string)
		res, err := s.client.SendDirectMessage(screenName, text)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to send direct message: %v", err)), nil
		}
		resBytes, _ := json.Marshal(res)
		return mcp.NewToolResultText(string(resBytes)), nil
	})

	// Tool: read_thread
	s.mcpServer.AddTool(mcp.NewTool("x_read_thread",
		mcp.WithDescription("Read a tweet thread and scroll through its replies and comments"),
		mcp.WithString("tweet_id", mcp.Required(), mcp.Description("ID of the parent tweet")),
		mcp.WithNumber("max_scrolls", mcp.Description("Number of scroll passes to fetch replies (default 4)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tweetID, _ := req.GetArguments()["tweet_id"].(string)
		maxScrolls := 4
		if ms, ok := req.GetArguments()["max_scrolls"].(float64); ok && ms > 0 {
			maxScrolls = int(ms)
		}

		details, err := s.client.ReadThread(tweetID, xspiderai.ScrollOptions{MaxScrolls: maxScrolls})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to read thread: %v", err)), nil
		}

		resBytes, _ := json.MarshalIndent(details, "", "  ")
		return mcp.NewToolResultText(string(resBytes)), nil
	})

	// Tool: search_tweets
	s.mcpServer.AddTool(mcp.NewTool("x_search_tweets",
		mcp.WithDescription("Search tweets on X and return matching tweets including media and metrics"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search keywords, hashtags, or query")),
		mcp.WithString("tab", mcp.Description("Search tab: 'Top' or 'Latest' (default 'Top')")),
		mcp.WithNumber("max_scrolls", mcp.Description("Number of scrolls to perform (default 3)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, _ := req.GetArguments()["query"].(string)
		tab, _ := req.GetArguments()["tab"].(string)
		maxScrolls := 3
		if ms, ok := req.GetArguments()["max_scrolls"].(float64); ok && ms > 0 {
			maxScrolls = int(ms)
		}

		results, err := s.client.SearchTweets(query, tab, xspiderai.ScrollOptions{MaxScrolls: maxScrolls})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to search tweets: %v", err)), nil
		}

		resBytes, _ := json.MarshalIndent(results, "", "  ")
		return mcp.NewToolResultText(string(resBytes)), nil
	})

	// Tool: discover
	s.mcpServer.AddTool(mcp.NewTool("x_discover",
		mcp.WithDescription("Multi-angle semantic discovery and research on X. Executes multiple complementary queries with temporal bounds (since/until), engagement filtering, reply inclusion, and sorting."),
		mcp.WithArray("queries", mcp.Required(), mcp.Description("List of query angles/phrasings to search in one coordinated call")),
		mcp.WithString("since", mcp.Description("Optional start date boundary in YYYY-MM-DD format")),
		mcp.WithString("until", mcp.Description("Optional end date boundary in YYYY-MM-DD format")),
		mcp.WithNumber("max_results", mcp.Description("Maximum number of tweets to return (default 20)")),
		mcp.WithString("sort", mcp.Description("Sort ordering: 'relevance' (default), 'recent', 'oldest', or 'engagement'")),
		mcp.WithNumber("min_engagement", mcp.Description("Minimum engagement floor (likes + retweets count)")),
		mcp.WithBoolean("include_replies", mcp.Description("Whether to include reply posts where answers/evidence often live (default false)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		var queries []string
		if rawQueries, ok := args["queries"].([]any); ok {
			for _, q := range rawQueries {
				if qs, ok := q.(string); ok && strings.TrimSpace(qs) != "" {
					queries = append(queries, strings.TrimSpace(qs))
				}
			}
		}

		since, _ := args["since"].(string)
		until, _ := args["until"].(string)
		sortVal, _ := args["sort"].(string)

		maxResults := 20
		if mr, ok := args["max_results"].(float64); ok && mr > 0 {
			maxResults = int(mr)
		}

		minEngagement := 0
		if me, ok := args["min_engagement"].(float64); ok && me > 0 {
			minEngagement = int(me)
		}

		includeReplies := false
		if ir, ok := args["include_replies"].(bool); ok {
			includeReplies = ir
		}

		results, err := s.client.Discover(xspiderai.DiscoverOptions{
			Queries:        queries,
			Since:          since,
			Until:          until,
			MaxResults:     maxResults,
			Sort:           sortVal,
			MinEngagement:  minEngagement,
			IncludeReplies: includeReplies,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to discover tweets: %v", err)), nil
		}

		resBytes, _ := json.MarshalIndent(results, "", "  ")
		return mcp.NewToolResultText(string(resBytes)), nil
	})

	// Tool: read_user_timeline
	s.mcpServer.AddTool(mcp.NewTool("x_read_user_timeline",
		mcp.WithDescription("Read recent tweets and posts from a user's profile timeline"),
		mcp.WithString("screen_name", mcp.Required(), mcp.Description("Target user's screen name / handle")),
		mcp.WithNumber("max_scrolls", mcp.Description("Number of scrolls to perform (default 3)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		screenName, _ := req.GetArguments()["screen_name"].(string)
		maxScrolls := 3
		if ms, ok := req.GetArguments()["max_scrolls"].(float64); ok && ms > 0 {
			maxScrolls = int(ms)
		}

		results, err := s.client.ReadUserTimeline(screenName, xspiderai.ScrollOptions{MaxScrolls: maxScrolls})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to read user timeline: %v", err)), nil
		}

		resBytes, _ := json.MarshalIndent(results, "", "  ")
		return mcp.NewToolResultText(string(resBytes)), nil
	})

	// Tool: scroll_page
	s.mcpServer.AddTool(mcp.NewTool("x_scroll_page",
		mcp.WithDescription("Perform a human-like scroll down on the current page to reveal and intercept more tweets"),
		mcp.WithNumber("scroll_count", mcp.Description("Number of scrolls (default 2)")),
		mcp.WithNumber("delay_ms", mcp.Description("Delay between scrolls in milliseconds (default 1500)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		scrollCount := 2
		if sc, ok := req.GetArguments()["scroll_count"].(float64); ok && sc > 0 {
			scrollCount = int(sc)
		}
		delayMs := 1500
		if d, ok := req.GetArguments()["delay_ms"].(float64); ok && d > 0 {
			delayMs = int(d)
		}

		results, err := s.client.Scroll(xspiderai.ScrollOptions{
			MaxScrolls: scrollCount,
			DelayMs:    delayMs,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to scroll: %v", err)), nil
		}

		resBytes, _ := json.MarshalIndent(results, "", "  ")
		return mcp.NewToolResultText(string(resBytes)), nil
	})

	// Tool: unfollow_user
	s.mcpServer.AddTool(mcp.NewTool("x_unfollow_user",
		mcp.WithDescription("Unfollow a user by screen name (@handle)"),
		mcp.WithString("screen_name", mcp.Required(), mcp.Description("Username / handle of the user")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		screenName, _ := req.GetArguments()["screen_name"].(string)
		res, err := s.client.UnfollowUser(screenName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to unfollow user: %v", err)), nil
		}
		resBytes, _ := json.Marshal(res)
		return mcp.NewToolResultText(string(resBytes)), nil
	})

	// Tool: get_profile
	s.mcpServer.AddTool(mcp.NewTool("x_get_profile",
		mcp.WithDescription("Get profile information and metrics for any user by their screen name"),
		mcp.WithString("screen_name", mcp.Required(), mcp.Description("Username / handle of the user")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		screenName, _ := req.GetArguments()["screen_name"].(string)
		prof, err := s.client.GetProfile(screenName)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get profile: %v", err)), nil
		}
		resBytes, _ := json.MarshalIndent(prof, "", "  ")
		return mcp.NewToolResultText(string(resBytes)), nil
	})

	// Tool: get_my_profile
	s.mcpServer.AddTool(mcp.NewTool("x_get_my_profile",
		mcp.WithDescription("Get the profile information and metrics for the currently authenticated account"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		prof, err := s.client.GetMyProfile()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get my profile: %v", err)), nil
		}
		resBytes, _ := json.MarshalIndent(prof, "", "  ")
		return mcp.NewToolResultText(string(resBytes)), nil
	})
}

// ServeStdio starts the MCP server over standard input/output
func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpServer)
}
