package chunker

import (
	"testing"
)

func TestSliceDateRangeMonthly(t *testing.T) {
	from := "01-01-2026"
	to := "01-04-2026"

	intervals, err := SliceDateRange(from, to, "monthly")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(intervals) != 3 {
		t.Fatalf("expected 3 intervals, got %d", len(intervals))
	}

	if intervals[0].From != "01-01-2026" || intervals[0].To != "01-02-2026" {
		t.Errorf("unexpected interval 0: %+v", intervals[0])
	}
	if intervals[2].From != "01-03-2026" || intervals[2].To != "01-04-2026" {
		t.Errorf("unexpected interval 2: %+v", intervals[2])
	}
}

func TestSliceDateRangeWeekly(t *testing.T) {
	from := "01-01-2026"
	to := "15-01-2026"

	intervals, err := SliceDateRange(from, to, "weekly")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(intervals) != 2 {
		t.Fatalf("expected 2 weekly intervals, got %d", len(intervals))
	}
}
