package notifier

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNotifierGenericWebhook(t *testing.T) {
	var receivedPayload Payload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	n := NewNotifier(Config{
		WebhookURL: server.URL,
	})

	if !n.Enabled() {
		t.Fatalf("notifier should be enabled")
	}

	payload := Payload{
		Status:      "completed",
		Query:       "golang test",
		TweetsSaved: 100,
		OutputFile:  "tweets-data/test.csv",
		Duration:    "1m30s",
	}

	if err := n.Notify(payload); err != nil {
		t.Fatalf("failed to dispatch notification: %v", err)
	}

	if receivedPayload.Query != "golang test" || receivedPayload.TweetsSaved != 100 {
		t.Errorf("unexpected received payload: %+v", receivedPayload)
	}
}
