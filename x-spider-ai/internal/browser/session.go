package browser

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
	"x-spider-ai/internal/models"
)

// SessionData stores Twitter cookies and extracted authorization tokens
type SessionData struct {
	AccountID   string            `json:"account_id,omitempty"`
	ScreenName  string            `json:"screen_name,omitempty"`
	AuthToken   string            `json:"auth_token"`
	CT0         string            `json:"ct0"`
	BearerToken string            `json:"bearer_token,omitempty"`
	CookieMap   map[string]string `json:"cookie_map,omitempty"`
	UserAgent   string            `json:"user_agent,omitempty"`
	UpdatedAt   string            `json:"updated_at,omitempty"`
}

// SessionManager manages persistent encrypted sessions inside an SQLite database
type SessionManager struct {
	dbPath    string
	db        *sql.DB
	secretKey []byte
}

const sessionTableSchema = `
CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	encrypted_payload TEXT NOT NULL,
	screen_name TEXT NOT NULL DEFAULT '',
	is_active INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS active_meta (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
`

// NewSessionManager initializes the SQLite-backed session store
func NewSessionManager(customDBPath string) *SessionManager {
	if customDBPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "."
		}
		dir := filepath.Join(homeDir, ".x-spider-ai")
		_ = os.MkdirAll(dir, 0700)
		customDBPath = filepath.Join(dir, "sessions.db")
	}

	key := deriveEncryptionKey()
	return &SessionManager{
		dbPath:    customDBPath,
		secretKey: key,
	}
}

func deriveEncryptionKey() []byte {
	hostname, _ := os.Hostname()
	user := os.Getenv("USERNAME")
	if user == "" {
		user = os.Getenv("USER")
	}
	seed := fmt.Sprintf("x-spider-ai-salt-2026-%s-%s", hostname, user)
	hash := sha256.Sum256([]byte(seed))
	return hash[:]
}

func (sm *SessionManager) initDB() error {
	if sm.db != nil {
		return nil
	}
	dir := filepath.Dir(sm.dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	db, err := sql.Open("sqlite", sm.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if _, err := db.Exec(sessionTableSchema); err != nil {
		_ = db.Close()
		return fmt.Errorf("failed to initialize sqlite sessions table: %w", err)
	}

	// Migrate existing legacy tables if columns missing
	_, _ = db.Exec("ALTER TABLE sessions ADD COLUMN screen_name TEXT NOT NULL DEFAULT '';")
	_, _ = db.Exec("ALTER TABLE sessions ADD COLUMN is_active INTEGER NOT NULL DEFAULT 0;")

	sm.db = db
	return nil
}

// SaveSession encrypts and stores the session inside the SQLite table under accountID
func (sm *SessionManager) SaveSession(data *SessionData) error {
	accountID := strings.TrimSpace(data.AccountID)
	if accountID == "" {
		accountID = "default"
	}
	data.AccountID = accountID

	if err := sm.initDB(); err != nil {
		return err
	}

	data.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	encrypted, err := sm.encrypt(payload)
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	query := `
	INSERT INTO sessions (id, encrypted_payload, screen_name, is_active, created_at, updated_at)
	VALUES (?, ?, ?, 1, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		encrypted_payload = excluded.encrypted_payload,
		screen_name = excluded.screen_name,
		is_active = 1,
		updated_at = excluded.updated_at;
	`
	// Mark others inactive and set this account active
	_, _ = sm.db.Exec("UPDATE sessions SET is_active = 0 WHERE id != ?", accountID)
	_, err = sm.db.Exec(query, accountID, encrypted, data.ScreenName, now, now)
	if err == nil {
		_, _ = sm.db.Exec("INSERT INTO active_meta (key, value) VALUES ('active_account', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", accountID)
	}
	return err
}

// LoadSession fetches and decrypts the currently active session from SQLite
func (sm *SessionManager) LoadSession() (*SessionData, error) {
	return sm.LoadAccountSession("")
}

// LoadAccountSession fetches and decrypts a specific account by ID or active account if accountID is empty
func (sm *SessionManager) LoadAccountSession(accountID string) (*SessionData, error) {
	if _, err := os.Stat(sm.dbPath); os.IsNotExist(err) {
		return nil, errors.New("sessions sqlite database does not exist")
	}

	if err := sm.initDB(); err != nil {
		return nil, err
	}

	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		// Look up active account from active_meta or sessions table
		var activeID string
		_ = sm.db.QueryRow("SELECT value FROM active_meta WHERE key = 'active_account'").Scan(&activeID)
		if activeID != "" {
			accountID = activeID
		} else {
			_ = sm.db.QueryRow("SELECT id FROM sessions WHERE is_active = 1 ORDER BY updated_at DESC LIMIT 1").Scan(&activeID)
			if activeID != "" {
				accountID = activeID
			} else {
				accountID = "default"
			}
		}
	}

	var encrypted string
	row := sm.db.QueryRow("SELECT encrypted_payload FROM sessions WHERE id = ? OR screen_name = ?", accountID, strings.TrimPrefix(accountID, "@"))
	if err := row.Scan(&encrypted); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("no session found in sqlite for account %q", accountID)
		}
		return nil, err
	}

	decrypted, err := sm.decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt session: %w", err)
	}

	var data SessionData
	if err := json.Unmarshal(decrypted, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// ListAccounts returns all registered account sessions in SQLite
func (sm *SessionManager) ListAccounts() ([]models.AccountInfo, error) {
	if _, err := os.Stat(sm.dbPath); os.IsNotExist(err) {
		return []models.AccountInfo{}, nil
	}

	if err := sm.initDB(); err != nil {
		return nil, err
	}

	var activeAccount string
	_ = sm.db.QueryRow("SELECT value FROM active_meta WHERE key = 'active_account'").Scan(&activeAccount)

	rows, err := sm.db.Query("SELECT id, screen_name, is_active, updated_at FROM sessions ORDER BY updated_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.AccountInfo
	for rows.Next() {
		var a models.AccountInfo
		var isActiveInt int
		if err := rows.Scan(&a.ID, &a.ScreenName, &isActiveInt, &a.UpdatedAt); err != nil {
			continue
		}
		a.IsActive = (isActiveInt == 1) || (a.ID == activeAccount)
		list = append(list, a)
	}

	return list, nil
}

// SwitchAccount sets a specific account ID as the default active account
func (sm *SessionManager) SwitchAccount(accountID string) error {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return fmt.Errorf("account ID cannot be empty")
	}

	if err := sm.initDB(); err != nil {
		return err
	}

	// Verify account exists
	var count int
	_ = sm.db.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = ? OR screen_name = ?", accountID, strings.TrimPrefix(accountID, "@")).Scan(&count)
	if count == 0 {
		return fmt.Errorf("account %q does not exist in sessions database", accountID)
	}

	_, _ = sm.db.Exec("UPDATE sessions SET is_active = 0")
	_, err := sm.db.Exec("UPDATE sessions SET is_active = 1 WHERE id = ? OR screen_name = ?", accountID, strings.TrimPrefix(accountID, "@"))
	if err != nil {
		return err
	}

	_, _ = sm.db.Exec("INSERT INTO active_meta (key, value) VALUES ('active_account', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", accountID)
	return nil
}

// DeleteAccount deletes a specific account session by ID
func (sm *SessionManager) DeleteAccount(accountID string) error {
	if err := sm.initDB(); err != nil {
		return err
	}
	_, err := sm.db.Exec("DELETE FROM sessions WHERE id = ? OR screen_name = ?", accountID, strings.TrimPrefix(accountID, "@"))
	return err
}

// ClearSession removes the active session record from the database
func (sm *SessionManager) ClearSession() error {
	if err := sm.initDB(); err != nil {
		return err
	}
	_, err := sm.db.Exec("DELETE FROM sessions WHERE is_active = 1 OR id = 'default'")
	_, _ = sm.db.Exec("DELETE FROM active_meta WHERE key = 'active_account'")
	return err
}

// DeleteDatabase closes connections and permanently removes the sessions.db file
func (sm *SessionManager) DeleteDatabase() error {
	if sm.db != nil {
		_ = sm.db.Close()
		sm.db = nil
	}
	if _, err := os.Stat(sm.dbPath); err == nil {
		return os.Remove(sm.dbPath)
	}
	return nil
}

// GetDBPath returns the filesystem path of the SQLite database
func (sm *SessionManager) GetDBPath() string {
	return sm.dbPath
}

func (sm *SessionManager) encrypt(plain []byte) (string, error) {
	block, err := aes.NewCipher(sm.secretKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	sealed := gcm.Seal(nonce, nonce, plain, nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func (sm *SessionManager) decrypt(encoded string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(sm.secretKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize {
		return nil, errors.New("invalid ciphertext")
	}

	nonce, ciphertext := raw[:nonceSize], raw[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
