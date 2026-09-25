package exporter

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
	"x-spider/internal/model"
)

func TestCSVExporter(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "xspider_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	csvPath := filepath.Join(tempDir, "test.csv")
	exp, err := NewCSVExporter(csvPath, "REPLACE")
	if err != nil {
		t.Fatalf("failed to create csv exporter: %v", err)
	}

	rows := []model.TweetRow{
		{
			ConversationIDStr: "1001",
			CreatedAt:         "2026-09-06T12:00:00.000Z",
			FavoriteCount:     10,
			FullText:          "Tweet content test",
			IDStr:             "1001",
			Username:          "tester",
		},
	}

	if err := exp.AppendRows(rows); err != nil {
		t.Fatalf("failed to append rows: %v", err)
	}
	if err := exp.Close(); err != nil {
		t.Fatalf("failed to close: %v", err)
	}

	// Verify CSV contents
	f, err := os.Open(csvPath)
	if err != nil {
		t.Fatalf("failed to open generated csv: %v", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to read csv records: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records (1 header + 1 row), got %d", len(records))
	}

	headers := records[0]
	if headers[0] != "conversation_id_str" {
		t.Errorf("expected first header conversation_id_str, got %s", headers[0])
	}
}

func TestExcelExporter(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "xspider_test_xlsx_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	xlsxPath := filepath.Join(tempDir, "test.xlsx")
	exp, err := NewExcelExporter(xlsxPath, "REPLACE")
	if err != nil {
		t.Fatalf("failed to create xlsx exporter: %v", err)
	}

	rows := []model.TweetRow{
		{
			ConversationIDStr: "2002",
			CreatedAt:         "2026-09-06T12:00:00.000Z",
			FavoriteCount:     99,
			FullText:          "Spreadsheet tweet",
			IDStr:             "2002",
			Username:          "xlsx_user",
		},
	}

	if err := exp.AppendRows(rows); err != nil {
		t.Fatalf("failed to append rows to xlsx: %v", err)
	}
	if err := exp.Close(); err != nil {
		t.Fatalf("failed to close: %v", err)
	}

	// Verify Excel file
	f, err := excelize.OpenFile(xlsxPath)
	if err != nil {
		t.Fatalf("failed to open generated xlsx: %v", err)
	}
	defer f.Close()

	sheetRows, err := f.GetRows("Tweets")
	if err != nil {
		t.Fatalf("failed to get sheet rows: %v", err)
	}

	if len(sheetRows) != 2 {
		t.Fatalf("expected 2 rows in sheet Tweets, got %d", len(sheetRows))
	}
}

func TestJSONExporter(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "xspider_test_json_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	jsonPath := filepath.Join(tempDir, "test.json")
	exp, err := NewJSONExporter(jsonPath, "REPLACE")
	if err != nil {
		t.Fatalf("failed to create json exporter: %v", err)
	}

	rows := []model.TweetRow{
		{
			ConversationIDStr: "3003",
			CreatedAt:         "2026-09-06T12:00:00.000Z",
			FavoriteCount:     50,
			FullText:          "JSON tweet content",
			IDStr:             "3003",
			Username:          "json_user",
		},
	}

	if err := exp.AppendRows(rows); err != nil {
		t.Fatalf("failed to append rows to json: %v", err)
	}
	if err := exp.Close(); err != nil {
		t.Fatalf("failed to close: %v", err)
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read generated json: %v", err)
	}

	var parsedRows []model.TweetRow
	if err := json.Unmarshal(data, &parsedRows); err != nil {
		t.Fatalf("failed to parse generated json: %v", err)
	}

	if len(parsedRows) != 1 {
		t.Fatalf("expected 1 row in json, got %d", len(parsedRows))
	}

	if parsedRows[0].IDStr != "3003" || parsedRows[0].Username != "json_user" {
		t.Errorf("unexpected parsed tweet: %+v", parsedRows[0])
	}
}

func TestJSONLExporter(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "xspider_test_jsonl_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	jsonlPath := filepath.Join(tempDir, "test.jsonl")
	exp, err := NewJSONLExporter(jsonlPath, "REPLACE")
	if err != nil {
		t.Fatalf("failed to create jsonl exporter: %v", err)
	}

	rows := []model.TweetRow{
		{IDStr: "4001", Username: "u1", FullText: "First line"},
		{IDStr: "4002", Username: "u2", FullText: "Second line"},
	}

	if err := exp.AppendRows(rows); err != nil {
		t.Fatalf("failed to append to jsonl: %v", err)
	}
	_ = exp.Close()

	data, err := os.ReadFile(jsonlPath)
	if err != nil {
		t.Fatalf("failed to read jsonl: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines in jsonl, got %d", len(lines))
	}
}

func TestSQLiteExporter(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "xspider_test_sqlite_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "tweets.db")
	exp, err := NewSQLiteExporter(dbPath, "REPLACE")
	if err != nil {
		t.Fatalf("failed to create sqlite exporter: %v", err)
	}

	rows := []model.TweetRow{
		{IDStr: "5001", Username: "sql_user", FullText: "Database tweet", CreatedAt: "2026-09-06T12:00:00.000Z"},
	}

	if err := exp.AppendRows(rows); err != nil {
		t.Fatalf("failed to append rows to sqlite: %v", err)
	}
	_ = exp.Close()

	// Verify querying SQLite
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM tweets WHERE id_str = ?", "5001").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query sqlite: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 record in sqlite, got %d", count)
	}
}

func TestMemoryExporter(t *testing.T) {
	exp := NewMemoryExporter()
	rows := []model.TweetRow{
		{IDStr: "6001", Username: "mem_user", FullText: "Memory tweet"},
	}
	if err := exp.AppendRows(rows); err != nil {
		t.Fatalf("failed to append rows to memory exporter: %v", err)
	}
	if len(exp.Rows()) != 1 {
		t.Fatalf("expected 1 row in memory exporter, got %d", len(exp.Rows()))
	}
	if exp.GetFilePath() != "(in-memory / webhook stream)" {
		t.Errorf("unexpected file path: %s", exp.GetFilePath())
	}
}


