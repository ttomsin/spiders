package exporter

import (
	"x-spider/internal/model"
)

// MemoryExporter retains scraped tweets in memory without writing to disk
type MemoryExporter struct {
	rows []model.TweetRow
}

// NewMemoryExporter creates an in-memory exporter (used for no-file webhook modes)
func NewMemoryExporter() *MemoryExporter {
	return &MemoryExporter{
		rows: make([]model.TweetRow, 0),
	}
}

// AppendRows appends tweets to the in-memory store
func (e *MemoryExporter) AppendRows(rows []model.TweetRow) error {
	e.rows = append(e.rows, rows...)
	return nil
}

// Close is a no-op for memory exporter
func (e *MemoryExporter) Close() error {
	return nil
}

// GetFilePath returns empty or virtual identifier
func (e *MemoryExporter) GetFilePath() string {
	return "(in-memory / webhook stream)"
}

// Rows returns all stored tweets
func (e *MemoryExporter) Rows() []model.TweetRow {
	return e.rows
}
