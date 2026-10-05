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
	"time"

	_ "modernc.org/sqlite"
)

// SessionData stores Twitter cookies and extracted authorization tokens
type SessionData struct {
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
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
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

	sm.db = db
	return nil
}

// SaveSession encrypts and stores the session inside the SQLite table
func (sm *SessionManager) SaveSession(data *SessionData) error {
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
	INSERT INTO sessions (id, encrypted_payload, created_at, updated_at)
	VALUES ('default', ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		encrypted_payload = excluded.encrypted_payload,
		updated_at = excluded.updated_at;
	`
	_, err = sm.db.Exec(query, encrypted, now, now)
	return err
}

// LoadSession fetches and decrypts the active session from SQLite
func (sm *SessionManager) LoadSession() (*SessionData, error) {
	if _, err := os.Stat(sm.dbPath); os.IsNotExist(err) {
		return nil, errors.New("sessions sqlite database does not exist")
	}

	if err := sm.initDB(); err != nil {
		return nil, err
	}

	var encrypted string
	row := sm.db.QueryRow("SELECT encrypted_payload FROM sessions WHERE id = 'default'")
	if err := row.Scan(&encrypted); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("no active session found in sqlite")
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

// ClearSession removes the active session record from the database
func (sm *SessionManager) ClearSession() error {
	if err := sm.initDB(); err != nil {
		return err
	}
	_, err := sm.db.Exec("DELETE FROM sessions WHERE id = 'default'")
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
