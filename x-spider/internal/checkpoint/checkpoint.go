package checkpoint

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

const DefaultCheckpointFile = ".xspider-checkpoint.json"

// State stores crawler execution progress for recovery
type State struct {
	Query        string `json:"query"`
	TargetFile   string `json:"target_file"`
	ExportFormat string `json:"export_format"`
	LastTweetID  string `json:"last_tweet_id"`
	TotalSaved   int    `json:"total_saved"`
	TargetLimit  int    `json:"target_limit"`
	UpdatedAt    string `json:"updated_at"`
}

// Manager handles saving, loading, and cleaning checkpoint state files
type Manager struct {
	filePath string
}

// NewManager creates a checkpoint manager with specified or default path
func NewManager(filePath string) *Manager {
	if filePath == "" {
		filePath = DefaultCheckpointFile
	}
	return &Manager{filePath: filePath}
}

// Exists checks if a checkpoint file is present on disk
func (m *Manager) Exists() bool {
	info, err := os.Stat(m.filePath)
	return err == nil && info.Size() > 0
}

// Save writes current crawler state to the checkpoint file
func (m *Manager) Save(state State) error {
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint state: %w", err)
	}
	return os.WriteFile(m.filePath, data, 0644)
}

// Load reads and parses saved checkpoint state
func (m *Manager) Load() (*State, error) {
	if !m.Exists() {
		return nil, errors.New("no checkpoint found")
	}
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read checkpoint file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse checkpoint json: %w", err)
	}
	return &state, nil
}

// Clear removes the checkpoint file upon successful crawl completion
func (m *Manager) Clear() error {
	if err := os.Remove(m.filePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
