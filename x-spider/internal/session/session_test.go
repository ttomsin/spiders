package session

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSessionStore(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	defer db.Close()

	store, err := NewCustomStore(db)
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	defer store.Close()

	sessionID := "job_ml_01"
	query := "machine learning"

	// 1. Create or get session
	rec, err := store.GetOrCreateSession(sessionID, query, "1000")
	if err != nil {
		t.Fatalf("failed to get/create session: %v", err)
	}
	if rec.SinceID != "1000" {
		t.Errorf("expected since_id 1000, got %s", rec.SinceID)
	}

	// 2. Record tweet batch
	tweets := []string{"2001", "2002", "2003"}
	if err := store.RecordTweets(sessionID, tweets, "2003"); err != nil {
		t.Fatalf("failed to record tweets: %v", err)
	}

	// 3. Check seen tweet IDs
	seen, err := store.GetSeenTweetIDs(sessionID)
	if err != nil {
		t.Fatalf("failed to get seen tweets: %v", err)
	}
	if len(seen) != 3 {
		t.Errorf("expected 3 seen tweets, got %d", len(seen))
	}
	if !seen["2002"] {
		t.Errorf("expected tweet 2002 to be marked seen")
	}

	// 4. Check HasTweet
	has, err := store.HasTweet(sessionID, "2001")
	if err != nil || !has {
		t.Errorf("expected HasTweet true for 2001")
	}
	hasNot, err := store.HasTweet(sessionID, "9999")
	if err != nil || hasNot {
		t.Errorf("expected HasTweet false for 9999")
	}

	// 5. Update session with newer sinceID
	recUpdated, err := store.GetOrCreateSession(sessionID, query, "2003")
	if err != nil {
		t.Fatalf("failed to retrieve updated session: %v", err)
	}
	if recUpdated.SinceID != "2003" {
		t.Errorf("expected since_id 2003, got %s", recUpdated.SinceID)
	}
	if recUpdated.TotalCount != 3 {
		t.Errorf("expected total_count 3, got %d", recUpdated.TotalCount)
	}
}
