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

// JSONLExporter writes scraped tweets as JSON Lines (one JSON object per line)
type JSONLExporter struct {
	mu         sync.Mutex
	filePath   string
	file       *os.File
	insertMode string
}

// NewJSONLExporter initializes a JSON Lines file exporter
func NewJSONLExporter(filePath string, insertMode string) (*JSONLExporter, error) {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if _, err := os.Stat(filePath); err == nil {
		if strings.ToUpper(insertMode) == "REPLACE" {
			oldPath := strings.TrimSuffix(filePath, ".jsonl") + ".old.jsonl"
			_ = os.Rename(filePath, oldPath)
		}
	}

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open jsonl file %s: %w", filePath, err)
	}

	return &JSONLExporter{
		filePath:   filePath,
		file:       f,
		insertMode: insertMode,
	}, nil
}

// AppendRows serializes each tweet onto its own line followed by \n
func (e *JSONLExporter) AppendRows(rows []model.TweetRow) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, row := range rows {
		lineBytes, err := json.Marshal(row)
		if err != nil {
			return fmt.Errorf("failed to marshal jsonl row: %w", err)
		}
		if _, err := e.file.Write(append(lineBytes, '\n')); err != nil {
			return fmt.Errorf("failed to write jsonl row: %w", err)
		}
	}

	return nil
}

// Close flushes and closes the JSONL file
func (e *JSONLExporter) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.file != nil {
		return e.file.Close()
	}
	return nil
}

// GetFilePath returns the file path
func (e *JSONLExporter) GetFilePath() string {
	return e.filePath
}
