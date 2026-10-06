package models

// MediaItem represents an image, video, or gif attached to a tweet
type MediaItem struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // "photo", "video", "animated_gif"
	MediaURL string `json:"media_url"`
	VideoURL string `json:"video_url,omitempty"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	Bitrate  int    `json:"bitrate,omitempty"`
}

// UserProfile represents Twitter/X user profile data
type UserProfile struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ScreenName      string `json:"screen_name"`
	Description     string `json:"description"`
	Location        string `json:"location"`
	FollowersCount  int    `json:"followers_count"`
	FriendsCount    int    `json:"friends_count"` // Following
	StatusesCount   int    `json:"statuses_count"`
	ProfileImageURL string `json:"profile_image_url"`
	Verified        bool   `json:"verified"`
}

// AccountInfo represents metadata about a stored Twitter/X account session
type AccountInfo struct {
	ID         string `json:"id"`
	ScreenName string `json:"screen_name,omitempty"`
	IsActive   bool   `json:"is_active"`
	UpdatedAt  string `json:"updated_at"`
}

// Tweet represents a full, rich tweet object with replies and media
type Tweet struct {
	ID                  string      `json:"id"`
	ConversationID      string      `json:"conversation_id"`
	FullText            string      `json:"full_text"`
	CreatedAt           string      `json:"created_at"`
	Author              UserProfile `json:"author"`
	InReplyToStatusID   string      `json:"in_reply_to_status_id,omitempty"`
	InReplyToScreenName string      `json:"in_reply_to_screen_name,omitempty"`
	QuotedStatusID      string      `json:"quoted_status_id,omitempty"`
	QuotedTweet         *Tweet      `json:"quoted_tweet,omitempty"`
	ReplyCount          int         `json:"reply_count"`
	RetweetCount        int         `json:"retweet_count"`
	FavoriteCount       int         `json:"favorite_count"`
	QuoteCount          int         `json:"quote_count"`
	BookmarkCount       int         `json:"bookmark_count"`
	ViewsCount          string      `json:"views_count,omitempty"`
	Media               []MediaItem `json:"media,omitempty"`
	TweetURL            string      `json:"tweet_url"`
	IsLiked             bool        `json:"is_liked,omitempty"`
	IsRetweeted         bool        `json:"is_retweeted,omitempty"`
	IsBookmarked        bool        `json:"is_bookmarked,omitempty"`
}

// ThreadDetails represents a tweet with its conversation thread and direct replies
type ThreadDetails struct {
	MainTweet Tweet   `json:"main_tweet"`
	Replies   []Tweet `json:"replies"`
}

// DirectMessage represents a direct message between users
type DirectMessage struct {
	ID        string      `json:"id"`
	Text      string      `json:"text"`
	SenderID  string      `json:"sender_id"`
	Recipient string      `json:"recipient"`
	CreatedAt string      `json:"created_at"`
	Media     []MediaItem `json:"media,omitempty"`
}

// ActionResult represents the standardized outcome of any action (like, post, follow, etc.)
type ActionResult struct {
	Success  bool   `json:"success"`
	Action   string `json:"action"`
	Target   string `json:"target,omitempty"`
	TweetID  string `json:"tweet_id,omitempty"`
	TweetURL string `json:"tweet_url,omitempty"`
	Data     any    `json:"data,omitempty"`
	Error    string `json:"error,omitempty"`
}

// PostTweetOptions allows passing optional parameters when posting/replying/quoting
type PostTweetOptions struct {
	Text            string   `json:"text"`
	InReplyToID     string   `json:"in_reply_to_id,omitempty"`
	QuoteTweetID    string   `json:"quote_tweet_id,omitempty"`
	MediaFilePaths  []string `json:"media_file_paths,omitempty"`
	PollOptions     []string `json:"poll_options,omitempty"`
	PollDurationMin int      `json:"poll_duration_min,omitempty"`
}

// ScrollOptions configures thread/timeline scrolling behavior
type ScrollOptions struct {
	MaxScrolls    int    `json:"max_scrolls"`
	DelayMs       int    `json:"delay_ms"`
	TargetCount   int    `json:"target_count"`
	ScrollElement string `json:"scroll_element,omitempty"`
}

// DiscoverOptions defines multi-angle semantic research parameters for agents
type DiscoverOptions struct {
	Queries        []string `json:"queries"`
	Since          string   `json:"since,omitempty"`          // e.g. YYYY-MM-DD
	Until          string   `json:"until,omitempty"`          // e.g. YYYY-MM-DD
	MaxResults     int      `json:"max_results,omitempty"`     // Result volume limit
	Sort           string   `json:"sort,omitempty"`           // "relevance", "recent", "oldest", "engagement"
	MinEngagement  int      `json:"min_engagement,omitempty"` // Minimum favorites/replies floor
	IncludeReplies bool     `json:"include_replies,omitempty"` // Whether to include reply posts
}
