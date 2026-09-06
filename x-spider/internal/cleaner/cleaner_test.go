package cleaner

import (
	"testing"
	"x-spider/internal/model"
)

func TestCleaner(t *testing.T) {
	c := NewCleaner(Options{
		StripURLs:     true,
		StripMentions: true,
		StripEmojis:   true,
		MinLength:     10,
	})

	row := model.TweetRow{
		FullText: "Hey @elonmusk check this out https://t.co/abc12345 🚀🔥 this is awesome text!",
	}

	cleaned, ok := c.Clean(row)
	if !ok {
		t.Fatalf("expected tweet to pass filter")
	}

	expected := "Hey check this out this is awesome text!"
	if cleaned.FullText != expected {
		t.Errorf("expected %q, got %q", expected, cleaned.FullText)
	}

	// Test min length rejection
	shortRow := model.TweetRow{
		FullText: "@someone hi 👍",
	}
	_, ok = c.Clean(shortRow)
	if ok {
		t.Errorf("expected short tweet to be filtered out, but it passed")
	}
}
