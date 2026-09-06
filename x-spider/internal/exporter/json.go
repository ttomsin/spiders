package exporter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"x-spider/internal/model"
)

// JSONExporter exports scraped tweets as a formatted JSON array
type JSONExporter struct {
	mu         sync.Mutex
	filePath   string
	rows       []model.TweetRow
	insertMode string
}

// NewJSONExporter creates a JSON file exporter with support for REPLACE/APPEND modes
func NewJSONExporter(filePath string, insertMode string) (*JSONExporter, error) {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	var initialRows []model.TweetRow

	if _, err := os.Stat(filePath); err == nil {
		if strings.ToUpper(insertMode) == "REPLACE" {
			oldPath := strings.TrimSuffix(filePath, ".json") + ".old.json"
			_ = os.Rename(filePath, oldPath)
		} else {
			// In APPEND mode, read existing tweets to preserve them
			if data, err := os.ReadFile(filePath); err == nil && len(data) > 0 {
				_ = json.Unmarshal(data, &initialRows)
			}
		}
	}

	return &JSONExporter{
		filePath:   filePath,
		rows:       initialRows,
		insertMode: insertMode,
	}, nil
}

// AppendRows appends rows and updates the JSON file on disk
func (e *JSONExporter) AppendRows(rows []model.TweetRow) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.rows = append(e.rows, rows...)

	data, err := json.MarshalIndent(e.rows, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}

	if err := os.WriteFile(e.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write json file %s: %w", e.filePath, err)
	}

	return nil
}

// Close finalizes the JSON exporter
func (e *JSONExporter) Close() error {
	return nil
}

// GetFilePath returns the file path
func (e *JSONExporter) GetFilePath() string {
	return e.filePath
}
