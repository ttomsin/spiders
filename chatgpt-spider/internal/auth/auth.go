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
	EncryptedSessionToken string            `json:"encrypted_session_token,omitempty"`
	Tokens                map[string]string `json:"tokens,omitempty"` // map cookie name -> encrypted token
	CreatedAt             string            `json:"created_at,omitempty"`
}

// GetBaseDir returns ~/.chatgpt-spider
func GetBaseDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(homeDir, ".chatgpt-spider")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// GetProfileDir returns ~/.chatgpt-spider/profile (for persistent browser sessions)
func GetProfileDir() (string, error) {
	base, err := GetBaseDir()
	if err != nil {
		return "", err
	}
	profileDir := filepath.Join(base, "profile")
	if err := os.MkdirAll(profileDir, 0700); err != nil {
		return "", err
	}
	return profileDir, nil
}

// getStoragePath returns ~/.chatgpt-spider/credentials.json
func getStoragePath() (string, error) {
	base, err := GetBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "credentials.json"), nil
}

func getMachineKey() []byte {
	hostname, _ := os.Hostname()
	user := os.Getenv("USERNAME")
	if user == "" {
		user = os.Getenv("USER")
	}
	seed := fmt.Sprintf("chatgpt-spider-salt-2026-%s-%s", hostname, user)
	hash := sha256.Sum256([]byte(seed))
	return hash[:]
}

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
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return string(plainText), nil
}

// SaveSessionCookies saves a map of cookie name -> value (e.g. .0 and .1)
func SaveSessionCookies(cookies map[string]string) error {
	encMap := make(map[string]string)
	for k, v := range cookies {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			enc, err := encrypt(trimmed)
			if err != nil {
				return err
			}
			encMap[k] = enc
		}
	}

	creds := StoredCredentials{
		Tokens: encMap,
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

// GetSessionCookies returns all decrypted cookie name -> value pairs
func GetSessionCookies() (map[string]string, error) {
	path, err := getStoragePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var creds StoredCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}

	result := make(map[string]string)
	if len(creds.Tokens) > 0 {
		for k, enc := range creds.Tokens {
			if dec, err := decrypt(enc); err == nil {
				result[k] = dec
			}
		}
	} else if creds.EncryptedSessionToken != "" {
		if dec, err := decrypt(creds.EncryptedSessionToken); err == nil {
			result["__Secure-next-auth.session-token"] = dec
		}
	}

	return result, nil
}

// SaveSessionToken securely saves the __Secure-next-auth.session-token (or chunked cookies)
func SaveSessionToken(token string) error {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return errors.New("token cannot be empty")
	}

	enc, err := encrypt(trimmed)
	if err != nil {
		return fmt.Errorf("failed to encrypt session token: %w", err)
	}

	creds := StoredCredentials{
		EncryptedSessionToken: enc,
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

// GetSessionToken returns the decrypted session token, if stored
func GetSessionToken() (string, error) {
	cookies, err := GetSessionCookies()
	if err != nil {
		return "", err
	}
	if len(cookies) == 0 {
		return "", nil
	}
	if tok, ok := cookies["__Secure-next-auth.session-token"]; ok {
		return tok, nil
	}
	if tok0, ok := cookies["__Secure-next-auth.session-token.0"]; ok {
		return tok0, nil
	}
	for _, v := range cookies {
		return v, nil
	}
	return "", nil
}

// ClearCredentials removes stored credentials
func ClearCredentials() error {
	path, err := getStoragePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// MaskToken masks a sensitive token
func MaskToken(token string) string {
	if len(token) <= 8 {
		return "********"
	}
	return fmt.Sprintf("%s...%s", token[:4], token[len(token)-4:])
}