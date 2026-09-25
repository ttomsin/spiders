package exporter

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"x-spider/internal/config"
	"x-spider/internal/model"
)

// Exporter defines the interface for saving scraped tweets
type Exporter interface {
	AppendRows(rows []model.TweetRow) error
	Close() error
	GetFilePath() string
}

// CSVExporter appends tweet rows into an RFC-4180 compliant CSV file
type CSVExporter struct {
	mu            sync.Mutex
	filePath      string
	file          *os.File
	writer        *csv.Writer
	headerWritten bool
}

// NewCSVExporter initializes a CSV file exporter, handling REPLACE/APPEND modes
func NewCSVExporter(filePath string, insertMode string) (*CSVExporter, error) {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Check existing file
	if _, err := os.Stat(filePath); err == nil {
		if strings.ToUpper(insertMode) == "REPLACE" {
			oldPath := strings.TrimSuffix(filePath, ".csv") + ".old.csv"
			_ = os.Rename(filePath, oldPath)
		}
	}

	// Determine if file exists and has content
	fileInfo, statErr := os.Stat(filePath)
	fileExists := statErr == nil && fileInfo.Size() > 0

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open csv file %s: %w", filePath, err)
	}

	w := csv.NewWriter(f)

	exporter := &CSVExporter{
		filePath:      filePath,
		file:          f,
		writer:        w,
		headerWritten: fileExists,
	}

	return exporter, nil
}

// Headers returns the sorted headers list
func Headers() []string {
	headers := make([]string, len(config.FilteredFields))
	copy(headers, config.FilteredFields)
	sort.Strings(headers)
	return headers
}

// AppendRows writes tweets to the CSV file
func (e *CSVExporter) AppendRows(rows []model.TweetRow) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	headers := Headers()

	if !e.headerWritten {
		if err := e.writer.Write(headers); err != nil {
			return fmt.Errorf("failed to write csv header: %w", err)
		}
		e.headerWritten = true
	}

	for _, row := range rows {
		rowMap := row.ToMap()
		record := make([]string, len(headers))
		for i, header := range headers {
			record[i] = rowMap[header]
		}
		if err := e.writer.Write(record); err != nil {
			return fmt.Errorf("failed to write csv row: %w", err)
		}
	}

	e.writer.Flush()
	return e.writer.Error()
}

// Close flushes and closes the underlying CSV file
func (e *CSVExporter) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.writer != nil {
		e.writer.Flush()
	}
	if e.file != nil {
		return e.file.Close()
	}
	return nil
}

// GetFilePath returns the destination file path
func (e *CSVExporter) GetFilePath() string {
	return e.filePath
}
