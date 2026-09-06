package session

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Store manages session tracking and tweet deduplication in ~/.x-spider/sessions.db
type Store struct {
	mu sync.Mutex
	db *sql.DB
}

// SessionRecord holds session metadata
type SessionRecord struct {
	SessionID  string
	Query      string
	SinceID    string
	TotalCount int
	CreatedAt  string
	UpdatedAt  string
}

// GetDBPath returns ~/.x-spider/sessions.db
func GetDBPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(homeDir, ".x-spider")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "sessions.db"), nil
}

const tableSchema = `
CREATE TABLE IF NOT EXISTS sessions (
	session_id TEXT PRIMARY KEY,
	query TEXT,
	since_id TEXT,
	total_count INTEGER DEFAULT 0,
	created_at TEXT,
	updated_at TEXT
);

CREATE TABLE IF NOT EXISTS session_tweets (
	session_id TEXT,
	tweet_id TEXT,
	created_at TEXT,
	PRIMARY KEY (session_id, tweet_id)
);

CREATE INDEX IF NOT EXISTS idx_session_tweets ON session_tweets(session_id);
`

// OpenStore opens or creates the SQLite session store
func OpenStore() (*Store, error) {
	dbPath, err := GetDBPath()
	if err != nil {
		return nil, fmt.Errorf("failed to determine sessions db path: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sessions db: %w", err)
	}

	if _, err := db.Exec(tableSchema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize session tables: %w", err)
	}

	return &Store{db: db}, nil
}

// NewCustomStore allows testing with in-memory or custom database instances
func NewCustomStore(db *sql.DB) (*Store, error) {
	if _, err := db.Exec(tableSchema); err != nil {
		return nil, fmt.Errorf("failed to initialize session tables: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// GetOrCreateSession retrieves an existing session or initializes a new one
func (s *Store) GetOrCreateSession(sessionID, query, sinceID string) (*SessionRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	var rec SessionRecord

	err := s.db.QueryRow(`
		SELECT session_id, query, since_id, total_count, created_at, updated_at
		FROM sessions WHERE session_id = ?
	`, sessionID).Scan(&rec.SessionID, &rec.Query, &rec.SinceID, &rec.TotalCount, &rec.CreatedAt, &rec.UpdatedAt)

	if err == sql.ErrNoRows {
		rec = SessionRecord{
			SessionID:  sessionID,
			Query:      query,
			SinceID:    sinceID,
			TotalCount: 0,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		_, err := s.db.Exec(`
			INSERT INTO sessions (session_id, query, since_id, total_count, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, rec.SessionID, rec.Query, rec.SinceID, rec.TotalCount, rec.CreatedAt, rec.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
		return &rec, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to query session: %w", err)
	}

	if sinceID != "" && sinceID != rec.SinceID {
		rec.SinceID = sinceID
		_, _ = s.db.Exec(`UPDATE sessions SET since_id = ?, updated_at = ? WHERE session_id = ?`, sinceID, now, sessionID)
	}

	return &rec, nil
}

// HasTweet checks if a tweet ID was already seen for this session
func (s *Store) HasTweet(sessionID, tweetID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var exists int
	err := s.db.QueryRow(`
		SELECT 1 FROM session_tweets WHERE session_id = ? AND tweet_id = ?
	`, sessionID, tweetID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// GetSeenTweetIDs returns all tweet IDs tracked for a session into a map for fast lookup
func (s *Store) GetSeenTweetIDs(sessionID string) (map[string]bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`SELECT tweet_id FROM session_tweets WHERE session_id = ?`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			seen[id] = true
		}
	}
	return seen, nil
}

// RecordTweets batches newly crawled tweet IDs into session_tweets and updates session metadata
func (s *Store) RecordTweets(sessionID string, tweetIDs []string, latestTweetID string) error {
	if len(tweetIDs) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339)
	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO session_tweets (session_id, tweet_id, created_at) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, id := range tweetIDs {
		if _, err := stmt.Exec(sessionID, id, now); err != nil {
			return err
		}
	}

	updateQuery := `
		UPDATE sessions
		SET total_count = total_count + ?,
		    since_id = CASE WHEN ? != '' THEN ? ELSE since_id END,
		    updated_at = ?
		WHERE session_id = ?
	`
	if _, err := tx.Exec(updateQuery, len(tweetIDs), latestTweetID, latestTweetID, now, sessionID); err != nil {
		return err
	}

	return tx.Commit()
}