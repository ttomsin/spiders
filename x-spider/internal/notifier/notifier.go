package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Config defines notification channels
type Config struct {
	DiscordWebhookURL string `yaml:"discord_webhook_url" json:"discord_webhook_url"`
	TelegramBotToken  string `yaml:"telegram_bot_token" json:"telegram_bot_token"`
	TelegramChatID    string `yaml:"telegram_chat_id" json:"telegram_chat_id"`
	WebhookURL        string `yaml:"webhook_url" json:"webhook_url"`
}

// Payload represents the summary data sent upon crawl completion or error
type Payload struct {
	Status      string `json:"status"` // "completed" or "error"
	Query       string `json:"query"`
	TweetsSaved int    `json:"tweets_saved"`
	OutputFile  string `json:"output_file"`
	Duration    string `json:"duration"`
	Error       string `json:"error,omitempty"`
}

// Notifier dispatches alerts across configured channels
type Notifier struct {
	cfg    Config
	client *http.Client
}

// NewNotifier creates a Notifier with standard HTTP timeout
func NewNotifier(cfg Config) *Notifier {
	return &Notifier{
		cfg: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Enabled returns true if at least one notification channel is configured
func (n *Notifier) Enabled() bool {
	return n.cfg.DiscordWebhookURL != "" ||
		(n.cfg.TelegramBotToken != "" && n.cfg.TelegramChatID != "") ||
		n.cfg.WebhookURL != ""
}

// Notify broadcasts the crawl summary payload to all configured channels
func (n *Notifier) Notify(p Payload) error {
	var errs []error

	if n.cfg.DiscordWebhookURL != "" {
		if err := n.sendDiscord(p); err != nil {
			errs = append(errs, fmt.Errorf("discord: %w", err))
		}
	}

	if n.cfg.TelegramBotToken != "" && n.cfg.TelegramChatID != "" {
		if err := n.sendTelegram(p); err != nil {
			errs = append(errs, fmt.Errorf("telegram: %w", err))
		}
	}

	if n.cfg.WebhookURL != "" {
		if err := n.sendGeneric(p); err != nil {
			errs = append(errs, fmt.Errorf("generic webhook: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("notification errors: %v", errs)
	}
	return nil
}

func (n *Notifier) sendDiscord(p Payload) error {
	color := 0x00ff00 // Green for completed
	if p.Status == "error" {
		color = 0xff0000 // Red for error
	}

	discordMsg := map[string]any{
		"embeds": []map[string]any{
			{
				"title":       fmt.Sprintf("x-spider Crawl: %s", stringsTitle(p.Status)),
				"color":       color,
				"description": fmt.Sprintf("**Query:** `%s`\n**Tweets Saved:** %d\n**Duration:** %s\n**Output:** `%s`", p.Query, p.TweetsSaved, p.Duration, p.OutputFile),
				"timestamp":   time.Now().UTC().Format(time.RFC3339),
			},
		},
	}

	data, err := json.Marshal(discordMsg)
	if err != nil {
		return err
	}

	resp, err := n.client.Post(n.cfg.DiscordWebhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("discord returned status %d", resp.StatusCode)
	}
	return nil
}

func (n *Notifier) sendTelegram(p Payload) error {
	text := fmt.Sprintf("🕷 *x-spider Crawl %s*\n\n*Query:* `%s`\n*Tweets Saved:* %d\n*Duration:* %s\n*Output:* `%s`",
		stringsTitle(p.Status), p.Query, p.TweetsSaved, p.Duration, p.OutputFile)
	if p.Error != "" {
		text += fmt.Sprintf("\n*Error:* %s", p.Error)
	}

	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.cfg.TelegramBotToken)
	form := url.Values{}
	form.Set("chat_id", n.cfg.TelegramChatID)
	form.Set("text", text)
	form.Set("parse_mode", "Markdown")

	resp, err := n.client.PostForm(endpoint, form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("telegram returned status %d", resp.StatusCode)
	}
	return nil
}

func (n *Notifier) sendGeneric(p Payload) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}

	resp, err := n.client.Post(n.cfg.WebhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}

func stringsTitle(s string) string {
	if s == "" {
		return ""
	}
	return fmt.Sprintf("%c%s", s[0]-32, s[1:])
}
