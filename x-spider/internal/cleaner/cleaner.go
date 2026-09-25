package cleaner

import (
	"regexp"
	"strings"
	"unicode"

	"x-spider/internal/model"
)

var (
	urlRegex     = regexp.MustCompile(`https?://\S+`)
	mentionRegex = regexp.MustCompile(`@[A-Za-z0-9_]+`)
	spaceRegex   = regexp.MustCompile(`\s+`)
)

// Options holds text normalization and filtering configurations
type Options struct {
	StripURLs     bool
	StripMentions bool
	StripEmojis   bool
	MinLength     int
}

// Cleaner applies NLP sanitization rules to tweet content
type Cleaner struct {
	opts Options
}

// NewCleaner creates a new Cleaner with the specified options
func NewCleaner(opts Options) *Cleaner {
	return &Cleaner{opts: opts}
}

// Clean processes a TweetRow. Returns the modified row and a boolean indicating whether it passed filters.
func (c *Cleaner) Clean(row model.TweetRow) (*model.TweetRow, bool) {
	text := row.FullText

	if c.opts.StripURLs {
		text = urlRegex.ReplaceAllString(text, "")
	}

	if c.opts.StripMentions {
		text = mentionRegex.ReplaceAllString(text, "")
	}

	if c.opts.StripEmojis {
		text = stripEmojis(text)
	}

	// Collapse multiple spaces and trim
	text = strings.TrimSpace(spaceRegex.ReplaceAllString(text, " "))

	// Check min length constraint
	if c.opts.MinLength > 0 && len([]rune(text)) < c.opts.MinLength {
		return nil, false
	}

	row.FullText = text
	return &row, true
}

// stripEmojis removes Unicode emojis and pictographic symbols
func stripEmojis(s string) string {
	var b strings.Builder
	for _, r := range s {
		// Check for common emoji ranges
		if isEmoji(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isEmoji(r rune) bool {
	// Emoticons, symbols, pictographs, transport, supplemental symbols
	if (r >= 0x1F600 && r <= 0x1F64F) || // Emoticons
		(r >= 0x1F300 && r <= 0x1F5FF) || // Misc Symbols and Pictographs
		(r >= 0x1F680 && r <= 0x1F6FF) || // Transport and Map
		(r >= 0x1F700 && r <= 0x1F77F) || // Alchemical Symbols
		(r >= 0x1F780 && r <= 0x1F7FF) || // Geometric Shapes Extended
		(r >= 0x1F800 && r <= 0x1F8FF) || // Supplemental Arrows-C
		(r >= 0x1F900 && r <= 0x1F9FF) || // Supplemental Symbols and Pictographs
		(r >= 0x1FA00 && r <= 0x1FA6F) || // Chess Symbols
		(r >= 0x1FA70 && r <= 0x1FAFF) || // Symbols and Pictographs Extended-A
		(r >= 0x2600 && r <= 0x26FF) || // Misc symbols
		(r >= 0x2700 && r <= 0x27BF) || // Dingbats
		(r >= 0xFE00 && r <= 0xFE0F) || // Variation Selectors
		(r >= 0x1F1E6 && r <= 0x1F1FF) { // Regional Indicator Symbols (flags)
		return true
	}
	return unicode.In(r, unicode.So) // Symbol, other
}
