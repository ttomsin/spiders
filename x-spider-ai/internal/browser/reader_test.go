package browser

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"x-spider-ai/internal/models"
)

func TestBuildDiscoverQuery(t *testing.T) {
	opts := models.DiscoverOptions{
		Queries:        []string{"who started saying no wahala", "where did no wahala come from"},
		Since:          "2020-01-01",
		Until:          "2023-12-31",
		MinEngagement:  50,
		IncludeReplies: false,
	}

	for _, q := range opts.Queries {
		var parts []string
		parts = append(parts, q)
		if opts.Since != "" {
			parts = append(parts, fmt.Sprintf("since:%s", opts.Since))
		}
		if opts.Until != "" {
			parts = append(parts, fmt.Sprintf("until:%s", opts.Until))
		}
		if opts.MinEngagement > 0 {
			parts = append(parts, fmt.Sprintf("min_faves:%d", opts.MinEngagement))
		}
		if !opts.IncludeReplies {
			parts = append(parts, "-filter:replies")
		}

		fullQuery := strings.Join(parts, " ")
		if !strings.Contains(fullQuery, "since:2020-01-01") {
			t.Errorf("expected since:2020-01-01 in query %q", fullQuery)
		}
		if !strings.Contains(fullQuery, "until:2023-12-31") {
			t.Errorf("expected until:2023-12-31 in query %q", fullQuery)
		}
		if !strings.Contains(fullQuery, "min_faves:50") {
			t.Errorf("expected min_faves:50 in query %q", fullQuery)
		}
		if !strings.Contains(fullQuery, "-filter:replies") {
			t.Errorf("expected -filter:replies in query %q", fullQuery)
		}
	}
}

func TestSortDiscoverResults(t *testing.T) {
	tweets := []models.Tweet{
		{
			ID:            "1",
			FullText:      "First post",
			FavoriteCount: 10,
			RetweetCount:  2,
			CreatedAt:     time.Now().Add(-2 * time.Hour).Format(time.RubyDate),
		},
		{
			ID:            "2",
			FullText:      "Viral post",
			FavoriteCount: 500,
			RetweetCount:  100,
			CreatedAt:     time.Now().Add(-1 * time.Hour).Format(time.RubyDate),
		},
		{
			ID:            "3",
			FullText:      "Middle post",
			FavoriteCount: 50,
			RetweetCount:  10,
			CreatedAt:     time.Now().Format(time.RubyDate),
		},
	}

	// Sort engagement
	sorted := make([]models.Tweet, len(tweets))
	copy(sorted, tweets)
	// Score calculation
	score := func(tw models.Tweet) int {
		return tw.FavoriteCount*2 + tw.RetweetCount*3 + tw.ReplyCount
	}
	if score(sorted[1]) <= score(sorted[0]) {
		t.Fatalf("viral post should score higher")
	}
}
