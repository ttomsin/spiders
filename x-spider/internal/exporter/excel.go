package exporter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xuri/excelize/v2"
	"x-spider/internal/model"
)

// ExcelExporter accumulates rows and writes out an XLSX spreadsheet
type ExcelExporter struct {
	mu         sync.Mutex
	filePath   string
	rows       []model.TweetRow
	insertMode string
}

// NewExcelExporter initializes an Excel file exporter
func NewExcelExporter(filePath string, insertMode string) (*ExcelExporter, error) {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if _, err := os.Stat(filePath); err == nil {
		if strings.ToUpper(insertMode) == "REPLACE" {
			oldPath := strings.TrimSuffix(filePath, ".xlsx") + ".old.xlsx"
			_ = os.Rename(filePath, oldPath)
		}
	}

	return &ExcelExporter{
		filePath:   filePath,
		rows:       make([]model.TweetRow, 0),
		insertMode: insertMode,
	}, nil
}

// AppendRows adds rows and saves the updated workbook to disk
func (e *ExcelExporter) AppendRows(rows []model.TweetRow) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.rows = append(e.rows, rows...)

	// Write out workbook
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Tweets"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}
	f.SetActiveSheet(index)
	_ = f.DeleteSheet("Sheet1") // remove default sheet

	headers := Headers()

	// Write header
	for colIdx, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		_ = f.SetCellValue(sheetName, cell, header)
	}

	// Write data rows
	for rowIdx, row := range e.rows {
		rowMap := row.ToMap()
		for colIdx, header := range headers {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			_ = f.SetCellValue(sheetName, cell, rowMap[header])
		}
	}

	if err := f.SaveAs(e.filePath); err != nil {
		return fmt.Errorf("failed to save xlsx to %s: %w", e.filePath, err)
	}

	return nil
}

// Close finalizes the Excel file
func (e *ExcelExporter) Close() error {
	return nil
}

// GetFilePath returns the file path
func (e *ExcelExporter) GetFilePath() string {
	return e.filePath
}
