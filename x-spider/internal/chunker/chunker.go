package chunker

import (
	"fmt"
	"strings"
	"time"
)

const DateLayout = "02-01-2006"

// Interval represents a date slice with From and To in DD-MM-YYYY format
type Interval struct {
	From string
	To   string
}

// SliceDateRange divides a timeframe into sequential monthly, weekly, or daily chunks
func SliceDateRange(fromStr, toStr, mode string) ([]Interval, error) {
	from, err := time.Parse(DateLayout, strings.TrimSpace(fromStr))
	if err != nil {
		return nil, fmt.Errorf("invalid from date (%s), expected DD-MM-YYYY: %w", fromStr, err)
	}

	to, err := time.Parse(DateLayout, strings.TrimSpace(toStr))
	if err != nil {
		return nil, fmt.Errorf("invalid to date (%s), expected DD-MM-YYYY: %w", toStr, err)
	}

	if from.After(to) {
		return nil, fmt.Errorf("from date (%s) cannot be after to date (%s)", fromStr, toStr)
	}

	mode = strings.ToLower(strings.TrimSpace(mode))
	var intervals []Interval
	curr := from

	for curr.Before(to) {
		var next time.Time
		switch mode {
		case "daily":
			next = curr.AddDate(0, 0, 1)
		case "weekly":
			next = curr.AddDate(0, 0, 7)
		default: // "monthly"
			next = curr.AddDate(0, 1, 0)
		}

		if next.After(to) {
			next = to
		}

		intervals = append(intervals, Interval{
			From: curr.Format(DateLayout),
			To:   next.Format(DateLayout),
		})

		curr = next
	}

	return intervals, nil
}
