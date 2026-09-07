package interceptor

import "testing"

func TestParseSSEStream(t *testing.T) {
	raw := []byte(`
data: {"conversation_id": "conv-123", "message": {"id": "m1", "author": {"role": "assistant"}, "content": {"parts": ["Hello "]}}}
data: {"conversation_id": "conv-123", "message": {"id": "m1", "author": {"role": "assistant"}, "content": {"parts": ["Hello world!"]}}}
data: [DONE]
`)

	text, convID, err := ParseSSEStream(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "Hello world!" {
		t.Errorf("expected 'Hello world!', got '%s'", text)
	}
	if convID != "conv-123" {
		t.Errorf("expected 'conv-123', got '%s'", convID)
	}
}