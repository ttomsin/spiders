package exporter

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
	"x-spider/internal/model"
)

// SQLiteExporter stores tweets directly into a local SQLite database table
type SQLiteExporter struct {
	mu         sync.Mutex
	filePath   string
	db         *sql.DB
	insertMode string
}

// NewSQLiteExporter creates or opens an SQLite database for tweet storage
func NewSQLiteExporter(filePath string, insertMode string) (*SQLiteExporter, error) {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if _, err := os.Stat(filePath); err == nil {
		if strings.ToUpper(insertMode) == "REPLACE" {
			oldPath := strings.TrimSuffix(filePath, ".db") + ".old.db"
			_ = os.Rename(filePath, oldPath)
		}
	}

	db, err := sql.Open("sqlite", filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS tweets (
		id_str TEXT PRIMARY KEY,
		conversation_id_str TEXT,
		created_at TEXT,
		favorite_count INTEGER,
		full_text TEXT,
		image_url TEXT,
		in_reply_to_screen_name TEXT,
		lang TEXT,
		location TEXT,
		quote_count INTEGER,
		reply_count INTEGER,
		retweet_count INTEGER,
		tweet_url TEXT,
		user_id_str TEXT,
		username TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_tweets_username ON tweets(username);
	CREATE INDEX IF NOT EXISTS idx_tweets_created_at ON tweets(created_at);
	`
	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize sqlite schema: %w", err)
	}

	return &SQLiteExporter{
		filePath:   filePath,
		db:         db,
		insertMode: insertMode,
	}, nil
}

// AppendRows inserts tweets in an atomic transaction using INSERT OR IGNORE / INSERT OR REPLACE
func (e *SQLiteExporter) AppendRows(rows []model.TweetRow) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	tx, err := e.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start sqlite transaction: %w", err)
	}
	defer tx.Rollback()

	insertOp := "INSERT OR IGNORE"
	if strings.ToUpper(e.insertMode) == "REPLACE" {
		insertOp = "INSERT OR REPLACE"
	}

	stmtSQL := fmt.Sprintf(`%s INTO tweets (
		id_str, conversation_id_str, created_at, favorite_count, full_text,
		image_url, in_reply_to_screen_name, lang, location, quote_count,
		reply_count, retweet_count, tweet_url, user_id_str, username
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, insertOp)

	stmt, err := tx.Prepare(stmtSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare sqlite statement: %w", err)
	}
	defer stmt.Close()

	for _, r := range rows {
		_, err := stmt.Exec(
			r.IDStr,
			r.ConversationIDStr,
			r.CreatedAt,
			r.FavoriteCount,
			r.FullText,
			r.ImageURL,
			r.InReplyToScreenName,
			r.Lang,
			r.Location,
			r.QuoteCount,
			r.ReplyCount,
			r.RetweetCount,
			r.TweetURL,
			r.UserIDStr,
			r.Username,
		)
		if err != nil {
			return fmt.Errorf("failed to insert tweet %s into sqlite: %w", r.IDStr, err)
		}
	}

	return tx.Commit()
}

// Close closes the database connection
func (e *SQLiteExporter) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.db != nil {
		return e.db.Close()
	}
	return nil
}

// GetFilePath returns the SQLite file path
func (e *SQLiteExporter) GetFilePath() string {
	return e.filePath
}
