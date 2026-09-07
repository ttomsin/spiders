package interceptor

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
)

// ConversationResponse represents the OpenAI conversation SSE JSON chunk
type ConversationResponse struct {
	Message struct {
		ID     string `json:"id"`
		Author struct {
			Role string `json:"role"`
		} `json:"author"`
		Content struct {
			ContentType string   `json:"content_type"`
			Parts       []string `json:"parts"`
		} `json:"content"`
		Status string `json:"status"`
	} `json:"message"`
	ConversationID string `json:"conversation_id"`
	Error          string `json:"error,omitempty"`
}

// ParseSSEStream parses Server-Sent Events from raw stream bytes and extracts the latest completed message text
func ParseSSEStream(rawBody []byte) (string, string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(rawBody))
	var latestText string
	var conversationID string

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		dataPayload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if dataPayload == "[DONE]" || dataPayload == "" {
			continue
		}

		var resp ConversationResponse
		if err := json.Unmarshal([]byte(dataPayload), &resp); err == nil {
			if resp.ConversationID != "" {
				conversationID = resp.ConversationID
			}
			if len(resp.Message.Content.Parts) > 0 {
				latestText = strings.Join(resp.Message.Content.Parts, "")
			}
		}
	}

	return latestText, conversationID, scanner.Err()
}