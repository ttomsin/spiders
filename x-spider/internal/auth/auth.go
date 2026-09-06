package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type StoredCredentials struct {
	EncryptedToken  string   `json:"encrypted_token,omitempty"`
	EncryptedTokens []string `json:"encrypted_tokens,omitempty"`
	CreatedAt       string   `json:"created_at,omitempty"`
}

// getStoragePath returns ~/.x-spider/credentials.json
func getStoragePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(homeDir, ".x-spider")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials.json"), nil
}

// getMachineKey generates an AES-256 encryption key derived from machine & user attributes
func getMachineKey() []byte {
	hostname, _ := os.Hostname()
	user := os.Getenv("USERNAME")
	if user == "" {
		user = os.Getenv("USER")
	}
	seed := fmt.Sprintf("x-spider-salt-2026-%s-%s", hostname, user)
	hash := sha256.Sum256([]byte(seed))
	return hash[:]
}

// encrypt encrypts plaintext using AES-GCM with machine-derived key
func encrypt(plainText string) (string, error) {
	key := getMachineKey()
	block, err := aes.NewCipher(key)
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

	cipherText := gcm.Seal(nonce, nonce, []byte(plainText), nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// decrypt decrypts ciphertext using AES-GCM with machine-derived key
func decrypt(cipherBase64 string) (string, error) {
	key := getMachineKey()
	data, err := base64.StdEncoding.DecodeString(cipherBase64)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, cipherText := data[:nonceSize], data[nonceSize:]
	plainText, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed (credential file may belong to a different machine): %w", err)
	}

	return string(plainText), nil
}

// SaveAuthTokens securely stores multiple tokens in the user's home directory
func SaveAuthTokens(tokens []string) error {
	var validTokens []string
	seen := make(map[string]bool)
	for _, t := range tokens {
		trimmed := strings.TrimSpace(t)
		if trimmed != "" && !seen[trimmed] {
			validTokens = append(validTokens, trimmed)
			seen[trimmed] = true
		}
	}

	if len(validTokens) == 0 {
		return errors.New("at least one non-empty token must be provided")
	}

	var encTokens []string
	for _, t := range validTokens {
		enc, err := encrypt(t)
		if err != nil {
			return fmt.Errorf("failed to encrypt token: %w", err)
		}
		encTokens = append(encTokens, enc)
	}

	creds := StoredCredentials{
		EncryptedTokens: encTokens,
	}

	path, err := getStoragePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// SaveAuthToken securely stores a single token (or replaces existing with single)
func SaveAuthToken(token string) error {
	return SaveAuthTokens([]string{token})
}

// AddAuthToken appends a new token to the stored credentials without removing existing ones
func AddAuthToken(token string) error {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return errors.New("token cannot be empty")
	}

	currentTokens, err := GetAuthTokens()
	if err != nil {
		return err
	}

	for _, existing := range currentTokens {
		if existing == trimmed {
			return errors.New("token already exists in pool")
		}
	}

	currentTokens = append(currentTokens, trimmed)
	return SaveAuthTokens(currentTokens)
}

// RemoveAuthToken removes a token at 1-based index
func RemoveAuthToken(index int) error {
	currentTokens, err := GetAuthTokens()
	if err != nil {
		return err
	}

	if index < 1 || index > len(currentTokens) {
		return fmt.Errorf("invalid index %d (stored tokens: %d)", index, len(currentTokens))
	}

	// Remove index-1
	currentTokens = append(currentTokens[:index-1], currentTokens[index:]...)
	if len(currentTokens) == 0 {
		return ClearAuthToken()
	}

	return SaveAuthTokens(currentTokens)
}

// GetAuthTokens retrieves and decrypts all stored auth tokens
func GetAuthTokens() ([]string, error) {
	path, err := getStoragePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No token stored
		}
		return nil, err
	}

	var creds StoredCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}

	var encryptedList []string
	if len(creds.EncryptedTokens) > 0 {
		encryptedList = creds.EncryptedTokens
	} else if creds.EncryptedToken != "" {
		encryptedList = []string{creds.EncryptedToken}
	}

	var decryptedList []string
	for _, enc := range encryptedList {
		dec, err := decrypt(enc)
		if err != nil {
			return nil, err
		}
		decryptedList = append(decryptedList, dec)
	}

	return decryptedList, nil
}

// GetAuthToken retrieves and decrypts the primary (first) stored auth token
func GetAuthToken() (string, error) {
	tokens, err := GetAuthTokens()
	if err != nil {
		return "", err
	}
	if len(tokens) == 0 {
		return "", nil
	}
	return tokens[0], nil
}

// ClearAuthToken removes the stored credentials file
func ClearAuthToken() error {
	path, err := getStoragePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// GetStorageFilePath returns the path where credentials are saved
func GetStorageFilePath() string {
	p, _ := getStoragePath()
	return p
}

// MaskToken masks a token for safe terminal display (e.g. 1a2b...89ef)
func MaskToken(token string) string {
	if len(token) <= 8 {
		return "********"
	}
	return fmt.Sprintf("%s...%s", token[:4], token[len(token)-4:])
}
