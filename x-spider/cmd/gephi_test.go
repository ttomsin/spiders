package cmd

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
)

func TestTransformCSVToGephi(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gephi_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	inputCSV := filepath.Join(tempDir, "input.csv")
	outputCSV := filepath.Join(tempDir, "output.csv")

	// Create sample input CSV
	inF, err := os.Create(inputCSV)
	if err != nil {
		t.Fatalf("failed to create input file: %v", err)
	}
	w := csv.NewWriter(inF)
	_ = w.Write([]string{"username", "in_reply_to_screen_name", "full_text"})
	_ = w.Write([]string{"alice", "bob", "Hello bob"})
	_ = w.Write([]string{"charlie", "", "Just a tweet"})
	_ = w.Write([]string{"", "", ""}) // should be ignored
	w.Flush()
	inF.Close()

	if err := TransformCSVToGephi(inputCSV, outputCSV); err != nil {
		t.Fatalf("TransformCSVToGephi failed: %v", err)
	}

	// Verify output
	outF, err := os.Open(outputCSV)
	if err != nil {
		t.Fatalf("failed to open output: %v", err)
	}
	defer outF.Close()

	r := csv.NewReader(outF)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("failed to read output csv: %v", err)
	}

	if len(records) != 3 { // 1 header + 2 rows
		t.Fatalf("expected 3 rows in output, got %d", len(records))
	}

	if records[0][0] != "source" || records[0][1] != "target" {
		t.Errorf("unexpected headers: %v", records[0])
	}
	if records[1][0] != "alice" || records[1][1] != "bob" {
		t.Errorf("unexpected row 1: %v", records[1])
	}
	if records[2][0] != "charlie" || records[2][1] != "" {
		t.Errorf("unexpected row 2: %v", records[2])
	}
}
