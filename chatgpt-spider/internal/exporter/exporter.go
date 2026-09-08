package exporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Turn represents one prompt and response pair
type Turn struct {
	Prompt         string `json:"prompt"`
	Response       string `json:"response"`
	ConversationID string `json:"conversation_id,omitempty"`
	Timestamp      string `json:"timestamp"`
}

// Conversation represents the full conversation record
type Conversation struct {
	Turns     []Turn `json:"turns"`
	CreatedAt string `json:"created_at"`
}

// SaveMarkdown writes conversation history to a clean markdown document
func SaveMarkdown(filePath string, turns []Turn) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	var buf bytes.Buffer
	buf.WriteString("# ChatGPT Conversation\n\n")

	for i, turn := range turns {
		buf.WriteString(fmt.Sprintf("## Prompt %d\n\n%s\n\n", i+1, turn.Prompt))
		buf.WriteString(fmt.Sprintf("## Response %d\n\n%s\n\n---\n\n", i+1, turn.Response))
	}

	return os.WriteFile(filePath, buf.Bytes(), 0644)
}

// SaveJSON writes conversation history to a formatted JSON document
func SaveJSON(filePath string, turns []Turn) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(turns, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// SaveRaw writes the raw response directly to a file (ideal for datasets, CSV, JSONL, code)
func SaveRaw(filePath string, content string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filePath, []byte(content), 0644)
}

// SendWebhook dispatches conversation data to an external HTTP POST webhook endpoint
func SendWebhook(webhookURL string, turns []Turn) error {
	if webhookURL == "" || len(turns) == 0 {
		return nil
	}

	payload := map[string]any{
		"event":     "chatgpt_response",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"count":     len(turns),
		"data":      turns,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook responded with status %d", resp.StatusCode)
	}

	return nil
}