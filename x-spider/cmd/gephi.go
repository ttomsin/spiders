package cmd

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	gephiInputPath  string
	gephiOutputPath string
)

// gephiCmd transforms a crawled tweet CSV into a Gephi-compatible edge list (source, target)
var gephiCmd = &cobra.Command{
	Use:   "gephi",
	Short: "Transform a crawled CSV to Gephi network graph edge format (source, target)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if gephiInputPath == "" || gephiOutputPath == "" {
			return fmt.Errorf("both --input (-i) and --output (-o) flags are required")
		}

		return TransformCSVToGephi(gephiInputPath, gephiOutputPath)
	},
}

// TransformCSVToGephi reads input CSV and writes source/target graph pairs
func TransformCSVToGephi(inputPath, outputPath string) error {
	inFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input CSV: %w", err)
	}
	defer inFile.Close()

	reader := csv.NewReader(inFile)
	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV headers: %w", err)
	}

	usernameIdx := -1
	replyIdx := -1

	for i, h := range headers {
		if h == "username" {
			usernameIdx = i
		}
		if h == "in_reply_to_screen_name" {
			replyIdx = i
		}
	}

	if usernameIdx == -1 {
		return fmt.Errorf("missing 'username' column in CSV")
	}

	// Prepare output
	outDir := filepath.Dir(outputPath)
	if outDir != "" && outDir != "." {
		_ = os.MkdirAll(outDir, 0755)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output CSV: %w", err)
	}
	defer outFile.Close()

	writer := csv.NewWriter(outFile)
	defer writer.Flush()

	// Write Gephi headers
	if err := writer.Write([]string{"source", "target"}); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	var rowCount int
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading CSV line: %w", err)
		}

		username := ""
		if usernameIdx < len(record) {
			username = record[usernameIdx]
		}

		replyTo := ""
		if replyIdx != -1 && replyIdx < len(record) {
			replyTo = record[replyIdx]
		}

		if username == "" && replyTo == "" {
			continue
		}

		if err := writer.Write([]string{username, replyTo}); err != nil {
			return fmt.Errorf("error writing record: %w", err)
		}
		rowCount++
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}

	fmt.Printf("CSV file was written successfully to %s (%d edges exported)\n", outputPath, rowCount)
	return nil
}

func init() {
	gephiCmd.Flags().StringVarP(&gephiInputPath, "input", "i", "", "Input CSV file path")
	gephiCmd.Flags().StringVarP(&gephiOutputPath, "output", "o", "", "Output CSV file path")
	_ = gephiCmd.MarkFlagRequired("input")
	_ = gephiCmd.MarkFlagRequired("output")

	RootCmd.AddCommand(gephiCmd)
}
