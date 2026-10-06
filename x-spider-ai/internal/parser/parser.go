package parser

import (
	"encoding/json"
	"strings"
	"time"

	"x-spider-ai/internal/models"
)

// ParseTwitterDate converts Twitter's legacy date format or ISO string to standard UTC ISO string
func ParseTwitterDate(raw string) string {
	if raw == "" {
		return ""
	}
	t, err := time.Parse("Mon Jan 02 15:04:05 -0700 2006", raw)
	if err != nil {
		return raw
	}
	return t.UTC().Format(time.RFC3339)
}

// ExtractTweetsFromGraphQL extracts all Tweet objects from a raw GraphQL timeline or search payload
func ExtractTweetsFromGraphQL(rawJSON []byte) []models.Tweet {
	var tweets []models.Tweet
	var root map[string]any
	if err := json.Unmarshal(rawJSON, &root); err != nil {
		return tweets
	}

	traverseForTweets(root, &tweets)
	return tweets
}

func traverseForTweets(node any, tweets *[]models.Tweet) {
	switch v := node.(type) {
	case map[string]any:
		// Look for tweet_results or itemContent or result with __typename == "Tweet"
		if typename, ok := v["__typename"].(string); ok && typename == "Tweet" {
			if tw := parseSingleTweet(v); tw != nil {
				*tweets = append(*tweets, *tw)
			}
			return
		}

		if res, ok := v["tweet_results"].(map[string]any); ok {
			if resMap, ok := res["result"].(map[string]any); ok {
				if tw := parseSingleTweet(resMap); tw != nil {
					*tweets = append(*tweets, *tw)
				}
			}
		}

		for _, val := range v {
			traverseForTweets(val, tweets)
		}
	case []any:
		for _, item := range v {
			traverseForTweets(item, tweets)
		}
	}
}

func parseSingleTweet(m map[string]any) *models.Tweet {
	// Might be wrapped in TweetWithVisibilityResults
	if typename, ok := m["__typename"].(string); ok && typename == "TweetWithVisibilityResults" {
		if tweetObj, ok := m["tweet"].(map[string]any); ok {
			m = tweetObj
		}
	}

	legacy, _ := m["legacy"].(map[string]any)
	if legacy == nil {
		return nil
	}

	id, _ := m["rest_id"].(string)
	if id == "" {
		if idVal, ok := legacy["id_str"].(string); ok {
			id = idVal
		}
	}
	if id == "" {
		return nil
	}

	fullText, _ := legacy["full_text"].(string)
	createdAt, _ := legacy["created_at"].(string)
	convID, _ := legacy["conversation_id_str"].(string)
	inReplyToStatusID, _ := legacy["in_reply_to_status_id_str"].(string)
	inReplyToScreenName, _ := legacy["in_reply_to_screen_name"].(string)
	isLiked, _ := legacy["favorited"].(bool)
	isRetweeted, _ := legacy["retweeted"].(bool)
	isBookmarked, _ := legacy["bookmarked"].(bool)

	replyCount := toInt(legacy["reply_count"])
	retweetCount := toInt(legacy["retweet_count"])
	favoriteCount := toInt(legacy["favorite_count"])
	quoteCount := toInt(legacy["quote_count"])
	bookmarkCount := toInt(legacy["bookmark_count"])

	var viewsStr string
	if views, ok := m["views"].(map[string]any); ok {
		if count, ok := views["count"].(string); ok {
			viewsStr = count
		}
	}

	// Parse User
	author := parseUser(m)

	// Parse Media
	mediaList := parseMedia(legacy)

	tw := &models.Tweet{
		ID:                  id,
		ConversationID:      convID,
		FullText:            fullText,
		CreatedAt:           ParseTwitterDate(createdAt),
		Author:              author,
		InReplyToStatusID:   inReplyToStatusID,
		InReplyToScreenName: inReplyToScreenName,
		ReplyCount:          replyCount,
		RetweetCount:        retweetCount,
		FavoriteCount:       favoriteCount,
		QuoteCount:          quoteCount,
		BookmarkCount:       bookmarkCount,
		ViewsCount:          viewsStr,
		Media:               mediaList,
		IsLiked:             isLiked,
		IsRetweeted:         isRetweeted,
		IsBookmarked:        isBookmarked,
	}

	if author.ScreenName != "" {
		tw.TweetURL = "https://x.com/" + author.ScreenName + "/status/" + id
	} else {
		tw.TweetURL = "https://x.com/i/status/" + id
	}

	// Check if this tweet quotes another tweet
	if quotedStatusID, ok := legacy["quoted_status_id_str"].(string); ok && quotedStatusID != "" {
		tw.QuotedStatusID = quotedStatusID
		if quotedResult, ok := m["quoted_status_result"].(map[string]any); ok {
			if qm, ok := quotedResult["result"].(map[string]any); ok {
				tw.QuotedTweet = parseSingleTweet(qm)
			}
		}
	}

	return tw
}

func parseUser(m map[string]any) models.UserProfile {
	var prof models.UserProfile

	// Core or legacy user
	core, _ := m["core"].(map[string]any)
	if core != nil {
		userObj := core["user_results"]
		if userObj == nil {
			userObj = core["user_result"]
		}

		if userMap, ok := userObj.(map[string]any); ok {
			res := userMap["result"]
			if resMap, ok := res.(map[string]any); ok {
				if restID, ok := resMap["rest_id"].(string); ok {
					prof.ID = restID
				}
				// It could be in resMap["legacy"] or directly in resMap
				uLegacy, _ := resMap["legacy"].(map[string]any)
				if uLegacy == nil {
					uLegacy = resMap
				}
				if uLegacy != nil {
					prof.Name, _ = uLegacy["name"].(string)
					prof.ScreenName, _ = uLegacy["screen_name"].(string)
					prof.Description, _ = uLegacy["description"].(string)
					prof.Location, _ = uLegacy["location"].(string)
					prof.FollowersCount = toInt(uLegacy["followers_count"])
					prof.FriendsCount = toInt(uLegacy["friends_count"])
					prof.StatusesCount = toInt(uLegacy["statuses_count"])
					prof.ProfileImageURL, _ = uLegacy["profile_image_url_https"].(string)
					prof.Verified, _ = uLegacy["verified"].(bool)
					return prof
				}
			}
		}
	}

	return prof
}

// ExtractUserFromGraphQL parses a user profile from GraphQL UserByScreenName or UserByRestId
func ExtractUserFromGraphQL(rawJSON []byte) *models.UserProfile {
	var root map[string]any
	if err := json.Unmarshal(rawJSON, &root); err != nil {
		return nil
	}

	var found *models.UserProfile
	var searchUser func(node any)
	searchUser = func(node any) {
		if found != nil {
			return
		}
		switch v := node.(type) {
		case map[string]any:
			if uLegacy, ok := v["legacy"].(map[string]any); ok {
				if screenName, ok := uLegacy["screen_name"].(string); ok && screenName != "" {
					prof := models.UserProfile{
						Name:            uLegacy["name"].(string),
						ScreenName:      screenName,
						Description:     uLegacy["description"].(string),
						Location:        uLegacy["location"].(string),
						FollowersCount:  toInt(uLegacy["followers_count"]),
						FriendsCount:    toInt(uLegacy["friends_count"]),
						StatusesCount:   toInt(uLegacy["statuses_count"]),
						ProfileImageURL: uLegacy["profile_image_url_https"].(string),
						Verified:        uLegacy["verified"].(bool),
					}
					if restID, ok := v["rest_id"].(string); ok {
						prof.ID = restID
					}
					found = &prof
					return
				}
			}
			for _, val := range v {
				searchUser(val)
			}
		case []any:
			for _, item := range v {
				searchUser(item)
			}
		}
	}

	searchUser(root)
	return found
}

func parseMedia(legacy map[string]any) []models.MediaItem {
	var list []models.MediaItem
	extended, ok := legacy["extended_entities"].(map[string]any)
	if !ok {
		// Fallback to entities
		extended, ok = legacy["entities"].(map[string]any)
		if !ok {
			return list
		}
	}

	mediaArray, ok := extended["media"].([]any)
	if !ok {
		return list
	}

	for _, item := range mediaArray {
		mMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		var m models.MediaItem
		m.ID, _ = mMap["id_str"].(string)
		m.Type, _ = mMap["type"].(string)
		m.MediaURL, _ = mMap["media_url_https"].(string)

		// Video info if video or gif
		if videoInfo, ok := mMap["video_info"].(map[string]any); ok {
			if variants, ok := videoInfo["variants"].([]any); ok {
				var maxBitrate int
				for _, v := range variants {
					varMap, ok := v.(map[string]any)
					if !ok {
						continue
					}
					contentType, _ := varMap["content_type"].(string)
					if !strings.Contains(contentType, "mp4") {
						continue
					}
					bitrate := toInt(varMap["bitrate"])
					url, _ := varMap["url"].(string)
					if bitrate >= maxBitrate && url != "" {
						maxBitrate = bitrate
						m.VideoURL = url
						m.Bitrate = bitrate
					}
				}
			}
		}

		if orig, ok := mMap["original_info"].(map[string]any); ok {
			m.Width = toInt(orig["width"])
			m.Height = toInt(orig["height"])
		}

		list = append(list, m)
	}

	return list
}

func toInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}
