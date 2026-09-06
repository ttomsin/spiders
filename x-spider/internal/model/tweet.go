package model

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// TwitterDateLayout is Twitter's legacy created_at format: "Mon Jan 02 15:04:05 -0700 2006"
const TwitterDateLayout = "Mon Jan 02 15:04:05 -0700 2006"

// FormatTwitterDate converts Twitter's date format to standard ISO 8601 (RFC3339 UTC)
func FormatTwitterDate(rawDate string) string {
	if rawDate == "" {
		return ""
	}
	t, err := time.Parse(TwitterDateLayout, rawDate)
	if err != nil {
		// If already in ISO or other format, return as is
		return rawDate
	}
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

// TweetRow represents the standardized 15-field output row
type TweetRow struct {
	ConversationIDStr   string `json:"conversation_id_str" csv:"conversation_id_str"`
	CreatedAt           string `json:"created_at" csv:"created_at"`
	FavoriteCount       int    `json:"favorite_count" csv:"favorite_count"`
	FullText            string `json:"full_text" csv:"full_text"`
	IDStr               string `json:"id_str" csv:"id_str"`
	ImageURL            string `json:"image_url" csv:"image_url"`
	InReplyToScreenName string `json:"in_reply_to_screen_name" csv:"in_reply_to_screen_name"`
	Lang                string `json:"lang" csv:"lang"`
	Location            string `json:"location" csv:"location"`
	QuoteCount          int    `json:"quote_count" csv:"quote_count"`
	ReplyCount          int    `json:"reply_count" csv:"reply_count"`
	RetweetCount        int    `json:"retweet_count" csv:"retweet_count"`
	TweetURL            string `json:"tweet_url" csv:"tweet_url"`
	UserIDStr           string `json:"user_id_str" csv:"user_id_str"`
	Username            string `json:"username" csv:"username"`
}

// ToMap returns the row fields as a key-value map for dynamic exporters
func (r *TweetRow) ToMap() map[string]string {
	return map[string]string{
		"conversation_id_str":     r.ConversationIDStr,
		"created_at":              r.CreatedAt,
		"favorite_count":          fmt.Sprintf("%d", r.FavoriteCount),
		"full_text":               r.FullText,
		"id_str":                  r.IDStr,
		"image_url":               r.ImageURL,
		"in_reply_to_screen_name": r.InReplyToScreenName,
		"lang":                    r.Lang,
		"location":                r.Location,
		"quote_count":             fmt.Sprintf("%d", r.QuoteCount),
		"reply_count":             fmt.Sprintf("%d", r.ReplyCount),
		"retweet_count":           fmt.Sprintf("%d", r.RetweetCount),
		"tweet_url":               r.TweetURL,
		"user_id_str":             r.UserIDStr,
		"username":                r.Username,
	}
}

// GraphQL Response structures

type UserLegacy struct {
	Name        string `json:"name"`
	ScreenName  string `json:"screen_name"`
	Location    string `json:"location"`
	Description string `json:"description"`
}

type UserResult struct {
	Legacy *UserLegacy `json:"legacy"`
	Core   *struct {
		Name       string `json:"name"`
		ScreenName string `json:"screen_name"`
	} `json:"core"`
}

type UserResultsWrapper struct {
	Result *UserResult `json:"result"`
}

type MediaEntity struct {
	MediaURLHTTPS string `json:"media_url_https"`
}

type UserMentionEntity struct {
	ScreenName string `json:"screen_name"`
}

type Entities struct {
	Media        []MediaEntity       `json:"media"`
	UserMentions []UserMentionEntity `json:"user_mentions"`
}

type TweetLegacy struct {
	BookmarkCount        int      `json:"bookmark_count"`
	Bookmarked           bool     `json:"bookmarked"`
	CreatedAt            string   `json:"created_at"`
	ConversationIDStr    string   `json:"conversation_id_str"`
	Entities             Entities `json:"entities"`
	FavoriteCount        int      `json:"favorite_count"`
	Favorited            bool     `json:"favorited"`
	FullText             string   `json:"full_text"`
	InReplyToScreenName  string   `json:"in_reply_to_screen_name"`
	InReplyToStatusIDStr string   `json:"in_reply_to_status_id_str"`
	InReplyToUserIDStr   string   `json:"in_reply_to_user_id_str"`
	IsQuoteStatus        bool     `json:"is_quote_status"`
	Lang                 string   `json:"lang"`
	QuoteCount           int      `json:"quote_count"`
	ReplyCount           int      `json:"reply_count"`
	RetweetCount         int      `json:"retweet_count"`
	Retweeted            bool     `json:"retweeted"`
	UserIDStr            string   `json:"user_id_str"`
	IDStr                string   `json:"id_str"`
}

type TweetResult struct {
	TypeName string       `json:"__typename"`
	RestID   string       `json:"rest_id"`
	Legacy   *TweetLegacy `json:"legacy"`
	Core     *struct {
		UserResults *UserResultsWrapper `json:"user_results"`
	} `json:"core"`
	Tweet *struct {
		Legacy *TweetLegacy `json:"legacy"`
		Core   *struct {
			UserResults *UserResultsWrapper `json:"user_results"`
		} `json:"core"`
	} `json:"tweet"`
}

type ItemContent struct {
	ItemType     string `json:"itemType"`
	TweetResults *struct {
		Result *TweetResult `json:"result"`
	} `json:"tweet_results"`
}

type TimelineItem struct {
	EntryID string `json:"entryId"`
	Item    *struct {
		ItemContent *ItemContent `json:"itemContent"`
	} `json:"item"`
}

type TimelineEntryContent struct {
	EntryType   string         `json:"entryType"`
	ItemContent *ItemContent   `json:"itemContent"`
	Items       []TimelineItem `json:"items"`
}

type TimelineEntry struct {
	EntryID string               `json:"entryId"`
	Content TimelineEntryContent `json:"content"`
}

type TimelineInstruction struct {
	Type    string          `json:"type"`
	Entries []TimelineEntry `json:"entries"`
}

// SearchTimelineResponse represents Twitter search API response
type SearchTimelineResponse struct {
	Data struct {
		SearchByRawQuery struct {
			SearchTimeline struct {
				Timeline struct {
					Instructions []TimelineInstruction `json:"instructions"`
				} `json:"timeline"`
			} `json:"search_timeline"`
		} `json:"search_by_raw_query"`
		ThreadedConversationWithInjectionsV2 struct {
			Instructions []TimelineInstruction `json:"instructions"`
		} `json:"threaded_conversation_with_injections_v2"`
	} `json:"data"`
}

// ExtractTweetsFromJSON parses SearchTimeline or TweetDetail JSON payload into TweetRows
func ExtractTweetsFromJSON(body []byte, isDetailMode bool) ([]TweetRow, error) {
	var resp SearchTimelineResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal twitter response: %w", err)
	}

	var instructions []TimelineInstruction
	if len(resp.Data.ThreadedConversationWithInjectionsV2.Instructions) > 0 {
		instructions = resp.Data.ThreadedConversationWithInjectionsV2.Instructions
	} else {
		instructions = resp.Data.SearchByRawQuery.SearchTimeline.Timeline.Instructions
	}

	if len(instructions) == 0 {
		return nil, nil
	}

	var entries []TimelineEntry
	for _, inst := range instructions {
		if len(inst.Entries) > 0 {
			entries = append(entries, inst.Entries...)
		}
	}

	var rows []TweetRow

	for _, entry := range entries {
		// skip promoted tweets
		if strings.Contains(strings.ToLower(entry.EntryID), "promoted") {
			continue
		}

		var tweetResult *TweetResult

		if isDetailMode {
			if len(entry.Content.Items) == 0 || entry.Content.Items[0].Item == nil || entry.Content.Items[0].Item.ItemContent == nil {
				continue
			}
			ic := entry.Content.Items[0].Item.ItemContent
			if ic.TweetResults == nil || ic.TweetResults.Result == nil {
				continue
			}
			tweetResult = ic.TweetResults.Result
		} else {
			if entry.Content.ItemContent == nil || entry.Content.ItemContent.TweetResults == nil {
				continue
			}
			tweetResult = entry.Content.ItemContent.TweetResults.Result
		}

		if tweetResult == nil {
			continue
		}

		// Resolve legacy tweet content
		var legacy *TweetLegacy
		if tweetResult.Legacy != nil {
			legacy = tweetResult.Legacy
		} else if tweetResult.Tweet != nil && tweetResult.Tweet.Legacy != nil {
			legacy = tweetResult.Tweet.Legacy
		}

		if legacy == nil {
			continue
		}

		// Resolve user result
		var userResult *UserResult
		if tweetResult.Core != nil && tweetResult.Core.UserResults != nil && tweetResult.Core.UserResults.Result != nil {
			userResult = tweetResult.Core.UserResults.Result
		} else if tweetResult.Tweet != nil && tweetResult.Tweet.Core != nil && tweetResult.Tweet.Core.UserResults != nil && tweetResult.Tweet.Core.UserResults.Result != nil {
			userResult = tweetResult.Tweet.Core.UserResults.Result
		}

		if userResult == nil {
			continue
		}

		// Extract username and location
		var username, location string
		if userResult.Core != nil && userResult.Core.ScreenName != "" {
			username = userResult.Core.ScreenName
		} else if userResult.Legacy != nil && userResult.Legacy.ScreenName != "" {
			username = userResult.Legacy.ScreenName
		}

		if userResult.Legacy != nil {
			location = userResult.Legacy.Location
		}

		// Image URL from first media item if available
		var imageURL string
		if len(legacy.Entities.Media) > 0 {
			imageURL = legacy.Entities.Media[0].MediaURLHTTPS
		}

		cleanText := legacy.FullText
		if isDetailMode && len(legacy.Entities.UserMentions) > 0 {
			firstWord := strings.Split(cleanText, " ")[0]
			replyTo := legacy.Entities.UserMentions[0].ScreenName
			if strings.HasPrefix(firstWord, "@") {
				cleanText = strings.TrimPrefix(cleanText, "@"+replyTo+" ")
			}
		}

		tweetURL := fmt.Sprintf("https://x.com/%s/status/%s", username, legacy.IDStr)

		row := TweetRow{
			ConversationIDStr:   legacy.ConversationIDStr,
			CreatedAt:           FormatTwitterDate(legacy.CreatedAt),
			FavoriteCount:       legacy.FavoriteCount,
			FullText:            cleanText,
			IDStr:               legacy.IDStr,
			ImageURL:            imageURL,
			InReplyToScreenName: legacy.InReplyToScreenName,
			Lang:                legacy.Lang,
			Location:            location,
			QuoteCount:          legacy.QuoteCount,
			ReplyCount:          legacy.ReplyCount,
			RetweetCount:        legacy.RetweetCount,
			TweetURL:            tweetURL,
			UserIDStr:           legacy.UserIDStr,
			Username:            username,
		}

		rows = append(rows, row)
	}

	return rows, nil
}
