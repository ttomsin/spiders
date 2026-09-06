package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"x-spider/internal/notifier"
)

const (
	DefaultSearchTab             = "TOP"
	DefaultExportFormat          = "csv"
	DefaultLimit                 = 10
	DefaultDelaySeconds          = 3
	DefaultDelayEvery100Tweets   = 10
	DefaultFolderDestination     = "./tweets-data"
	DefaultSearchAdvancedURLTop  = "https://x.com/search-advanced"
	DefaultSearchAdvancedURLLive = "https://x.com/search-advanced?f=live"
)

var FilteredFields = []string{
	"conversation_id_str",
	"created_at",
	"favorite_count",
	"full_text",
	"id_str",
	"image_url",
	"in_reply_to_screen_name",
	"lang",
	"location",
	"quote_count",
	"reply_count",
	"retweet_count",
	"tweet_url",
	"user_id_str",
	"username",
}

// Config holds all runtime configurations for x-spider
type Config struct {
	AuthToken                string          `yaml:"auth_token" json:"auth_token"`
	AuthTokens               []string        `yaml:"auth_tokens" json:"auth_tokens"` // Token pool
	SearchKeyword            string          `yaml:"search_keyword" json:"search_keyword"`
	ThreadURL                string          `yaml:"thread_url" json:"thread_url"`
	FromDate                 string          `yaml:"from_date" json:"from_date"` // DD-MM-YYYY
	ToDate                   string          `yaml:"to_date" json:"to_date"`     // DD-MM-YYYY
	Chunk                    string          `yaml:"chunk" json:"chunk"`         // monthly, weekly, daily
	Resume                   bool            `yaml:"resume" json:"resume"`       // resume from checkpoint
	Limit                    int             `yaml:"limit" json:"limit"`
	DelaySeconds             int             `yaml:"delay_seconds" json:"delay_seconds"`
	DelayEvery100Tweets      int             `yaml:"delay_every_100_tweets" json:"delay_every_100_tweets"`
	OutputFilename           string          `yaml:"output_filename" json:"output_filename"`
	SearchTab                string          `yaml:"search_tab" json:"search_tab"`           // TOP or LATEST
	ExportFormat             string          `yaml:"export_format" json:"export_format"`     // csv, xlsx, json, jsonl, sqlite
	CSVInsertMode            string          `yaml:"csv_insert_mode" json:"csv_insert_mode"` // REPLACE or APPEND
	FolderDestination        string          `yaml:"folder_destination" json:"folder_destination"`
	Headless                 bool            `yaml:"headless" json:"headless"`
	Debug                    bool            `yaml:"debug" json:"debug"`
	EnableExponentialBackoff bool            `yaml:"enable_exponential_backoff" json:"enable_exponential_backoff"`
	Proxy                    string          `yaml:"proxy" json:"proxy"`
	Proxies                  []string        `yaml:"proxies" json:"proxies"` // Proxy pool
	BlockedExtensions        []string        `yaml:"blocked_extensions" json:"blocked_extensions"`
	StripURLs                bool            `yaml:"strip_urls" json:"strip_urls"`
	StripMentions            bool            `yaml:"strip_mentions" json:"strip_mentions"`
	StripEmojis              bool            `yaml:"strip_emojis" json:"strip_emojis"`
	MinLength                int             `yaml:"min_length" json:"min_length"`
	NoFile                   bool            `yaml:"no_file" json:"no_file"`                   // Stream/webhook only, skip saving to disk
	WebhookData              bool            `yaml:"webhook_data" json:"webhook_data"`         // Include crawled tweet records in webhook payload
	SessionID                string          `yaml:"session_id" json:"session_id"`             // Persistent session identifier for tracking/deduplication in SQLite
	SinceID                  string          `yaml:"since_id" json:"since_id"`                 // Lower bound tweet ID filter
	Notifications            notifier.Config `yaml:"notifications" json:"notifications"`
}

// NewDefaultConfig returns a configuration struct with standard defaults
func NewDefaultConfig() *Config {
	return &Config{
		Limit:                    DefaultLimit,
		DelaySeconds:             DefaultDelaySeconds,
		DelayEvery100Tweets:      DefaultDelayEvery100Tweets,
		SearchTab:                DefaultSearchTab,
		ExportFormat:             DefaultExportFormat,
		CSVInsertMode:            "REPLACE",
		FolderDestination:        DefaultFolderDestination,
		Headless:                 true,
		Debug:                    false,
		EnableExponentialBackoff: false,
		BlockedExtensions: []string{
			".jpg", ".jpeg", ".png", ".gif", ".webp", ".mp4", ".ts", "format=jpg",
		},
	}
}

// LoadConfigFile loads YAML configuration from the given filepath
func LoadConfigFile(path string) (*Config, error) {
	cfg := NewDefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml config at %s: %w", path, err)
	}

	return cfg, nil
}

// ApplyEnvOverrides overrides settings using environment variables if present
func (c *Config) ApplyEnvOverrides() {
	if val := os.Getenv("DEV_ACCESS_TOKEN"); val != "" && c.AuthToken == "" {
		c.AuthToken = val
	}
	if val := os.Getenv("TWITTER_AUTH_TOKEN"); val != "" && c.AuthToken == "" {
		c.AuthToken = val
	}
	if val := os.Getenv("HEADLESS_MODE"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			c.Headless = b
		}
	}
	if val := os.Getenv("ENABLE_EXPONENTIAL_BACKOFF"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			c.EnableExponentialBackoff = b
		}
	}
	if val := os.Getenv("DEBUG_MODE"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			c.Debug = b
		}
	}
	if val := os.Getenv("PROXY"); val != "" {
		c.Proxy = val
	}
	if val := os.Getenv("NO_FILE"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			c.NoFile = b
		}
	}
	if val := os.Getenv("WEBHOOK_DATA"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			c.WebhookData = b
		}
	}
	if val := os.Getenv("WEBHOOK_URL"); val != "" && c.Notifications.WebhookURL == "" {
		c.Notifications.WebhookURL = val
	}
	if val := os.Getenv("SESSION_ID"); val != "" && c.SessionID == "" {
		c.SessionID = val
	}
	if val := os.Getenv("SINCE_ID"); val != "" && c.SinceID == "" {
		c.SinceID = val
	}
}

// Normalize validates and normalizes field values
func (c *Config) Normalize() {
	if c.WebhookData {
		c.Notifications.WebhookIncludeData = true
	}
	if c.Notifications.WebhookIncludeData {
		c.WebhookData = true
	}

	c.SearchTab = strings.ToUpper(strings.TrimSpace(c.SearchTab))
	if c.SearchTab != "LATEST" && c.SearchTab != "TOP" {
		c.SearchTab = "TOP"
	}

	c.ExportFormat = strings.ToLower(strings.TrimSpace(c.ExportFormat))
	validFormats := map[string]bool{"csv": true, "xlsx": true, "json": true, "jsonl": true, "sqlite": true}
	if !validFormats[c.ExportFormat] {
		c.ExportFormat = "csv"
	}

	c.CSVInsertMode = strings.ToUpper(strings.TrimSpace(c.CSVInsertMode))
	if c.CSVInsertMode != "APPEND" {
		c.CSVInsertMode = "REPLACE"
	}

	if c.FolderDestination == "" {
		c.FolderDestination = DefaultFolderDestination
	}

	if c.DelaySeconds <= 0 {
		c.DelaySeconds = DefaultDelaySeconds
	}
	if c.Limit <= 0 {
		c.Limit = DefaultLimit
	}
}

// BuildTargetFilePath computes the full output file path based on config and timestamp
func (c *Config) BuildTargetFilePath() (string, error) {
	ext := c.ExportFormat
	if ext == "sqlite" {
		ext = "db"
	}

	nowStr := time.Now().Format("02-01-2006 15-04-05")

	var baseName string
	if c.OutputFilename != "" {
		baseName = strings.TrimSpace(c.OutputFilename)
		baseName = strings.TrimSuffix(baseName, ".csv")
		baseName = strings.TrimSuffix(baseName, ".xlsx")
		baseName = strings.TrimSuffix(baseName, ".json")
		baseName = strings.TrimSuffix(baseName, ".jsonl")
		baseName = strings.TrimSuffix(baseName, ".db")
	} else {
		keyword := c.SearchKeyword
		if keyword == "" && c.ThreadURL != "" {
			parts := strings.Split(c.ThreadURL, "/")
			keyword = "thread_" + parts[len(parts)-1]
		}
		baseName = fmt.Sprintf("%s %s", keyword, nowStr)
	}

	// replace spaces with underscores, colons with hyphens
	baseName = strings.ReplaceAll(baseName, " ", "_")
	baseName = strings.ReplaceAll(baseName, ":", "-")

	fileName := fmt.Sprintf("%s.%s", baseName, ext)
	return filepath.Join(c.FolderDestination, fileName), nil
}

// GetSearchURL returns the starting URL for the chosen search tab
func (c *Config) GetSearchURL() string {
	if c.ThreadURL != "" {
		return c.ThreadURL
	}
	if c.SearchTab == "LATEST" {
		return DefaultSearchAdvancedURLLive
	}
	return DefaultSearchAdvancedURLTop
}

// GetSwitchedSearchURL returns the fallback URL for the alternative search tab
func (c *Config) GetSwitchedSearchURL() (string, string) {
	if c.SearchTab == "TOP" {
		return "LATEST", DefaultSearchAdvancedURLLive
	}
	return "TOP", DefaultSearchAdvancedURLTop
}
