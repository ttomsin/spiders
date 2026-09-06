package model

import (
	"testing"
)

func TestFormatTwitterDate(t *testing.T) {
	raw := "Wed Oct 10 20:19:24 +0000 2018"
	expected := "2018-10-10T20:19:24.000Z"
	actual := FormatTwitterDate(raw)
	if actual != expected {
		t.Errorf("expected %s, got %s", expected, actual)
	}
}

func TestExtractTweetsFromJSON(t *testing.T) {
	sampleJSON := []byte(`{
		"data": {
			"search_by_raw_query": {
				"search_timeline": {
					"timeline": {
						"instructions": [
							{
								"type": "TimelineAddEntries",
								"entries": [
									{
										"entryId": "tweet-123456789",
										"content": {
											"entryType": "TimelineTimelineItem",
											"itemContent": {
												"itemType": "TimelineTweet",
												"tweet_results": {
													"result": {
														"__typename": "Tweet",
														"rest_id": "123456789",
														"legacy": {
															"id_str": "123456789",
															"conversation_id_str": "123456789",
															"created_at": "Sun Sep 06 12:00:00 +0000 2026",
															"full_text": "Hello world from Go x-spider!",
															"favorite_count": 42,
															"retweet_count": 10,
															"reply_count": 5,
															"quote_count": 2,
															"lang": "en",
															"user_id_str": "987654321",
															"entities": {
																"media": [
																	{"media_url_https": "https://pbs.twimg.com/media/sample.jpg"}
																]
															}
														},
														"core": {
															"user_results": {
																"result": {
																	"legacy": {
																		"screen_name": "gopher_dev",
																		"location": "Cyberspace"
																	}
																}
															}
														}
													}
												}
											}
										}
									},
									{
										"entryId": "promoted-tweet-99999",
										"content": {
											"itemContent": {}
										}
									}
								]
							}
						]
					}
				}
			}
		}
	}`)

	rows, err := ExtractTweetsFromJSON(sampleJSON, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("expected 1 tweet row (excluding promoted), got %d", len(rows))
	}

	row := rows[0]
	if row.IDStr != "123456789" {
		t.Errorf("expected IDStr 123456789, got %s", row.IDStr)
	}
	if row.Username != "gopher_dev" {
		t.Errorf("expected username gopher_dev, got %s", row.Username)
	}
	if row.TweetURL != "https://x.com/gopher_dev/status/123456789" {
		t.Errorf("expected url https://x.com/gopher_dev/status/123456789, got %s", row.TweetURL)
	}
	if row.FavoriteCount != 42 {
		t.Errorf("expected favorite count 42, got %d", row.FavoriteCount)
	}
	if row.ImageURL != "https://pbs.twimg.com/media/sample.jpg" {
		t.Errorf("expected image url, got %s", row.ImageURL)
	}
	if row.Location != "Cyberspace" {
		t.Errorf("expected location Cyberspace, got %s", row.Location)
	}
}
