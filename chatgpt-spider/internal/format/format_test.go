package format

import (
	"strings"
	"testing"
)

func TestApplyFormatConstraint(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"json", "valid, parseable raw JSON"},
		{"csv", "raw CSV format with a valid header row"},
		{"xml", "valid, well-formed XML"},
		{"yaml", "valid YAML"},
		{"markdown", "clean GitHub-Flavored Markdown"},
		{"custom schema", "custom schema"},
	}

	for _, tc := range tests {
		res := ApplyFormatConstraint("Hello world", tc.format)
		if !strings.Contains(res, tc.expected) {
			t.Errorf("expected format %q to contain %q, got %s", tc.format, tc.expected, res)
		}
	}

	// Empty format should return original prompt unmodified
	if res := ApplyFormatConstraint("Hello world", ""); res != "Hello world" {
		t.Errorf("expected unmodified prompt for empty format, got %s", res)
	}
}
