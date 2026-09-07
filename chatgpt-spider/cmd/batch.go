package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"chatgpt-spider/internal/exporter"
	"chatgpt-spider/internal/spider"
)

var (
	inputFile string
	delaySec  int
)

var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Execute a batch list of prompts from a text file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if inputFile == "" {
			return fmt.Errorf("--input / -i is required")
		}

		file, err := os.Open(inputFile)
		if err != nil {
			return fmt.Errorf("failed to open input file: %w", err)
		}
		defer file.Close()

		var prompts []string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" && !strings.HasPrefix(line, "#") {
				prompts = append(prompts, line)
			}
		}

		if len(prompts) == 0 {
			return fmt.Errorf("no prompts found in %s", inputFile)
		}

		cyan := color.New(color.FgCyan).SprintfFunc()
		green := color.New(color.FgGreen).SprintfFunc()
		yellow := color.New(color.FgYellow).SprintfFunc()

		fmt.Printf("%s\n", cyan("Loaded %d prompts from %s", len(prompts), inputFile))
		fmt.Printf("%s\n", yellow("Launching browser..."))

		eng, err := spider.NewEngine(spider.Options{
			Headless:     headless,
			Anonymous:    anon,
			SessionToken: sessionToken,
			Debug:        debug,
		})
		if err != nil {
			return err
		}
		defer eng.Close()

		if chatSessionID != "" {
			fmt.Printf("%s\n", yellow("Resuming conversation: %s...", chatSessionID))
		}

		if err := eng.Initialize(chatSessionID); err != nil {
			return err
		}

		var turns []exporter.Turn

		for i, promptText := range prompts {
			fmt.Printf("\n%s\n", cyan("━━━ Processing [%d/%d]: %s ━━━", i+1, len(prompts), promptText))

			resp, err := eng.Prompt(cmd.Context(), spider.PromptRequest{
				Prompt: promptText,
				Format: formatSpec,
			})
			if err != nil {
				fmt.Printf("%s\n", color.RedString("Error: %v", err))
				continue
			}

			fmt.Printf("%s\n", green("Response captured (%d characters)", len(resp.Text)))

			turn := exporter.Turn{
				Prompt:         promptText,
				Response:       resp.Text,
				ConversationID: resp.ConversationID,
				Timestamp:      resp.Timestamp,
			}
			turns = append(turns, turn)

			if webhookURL != "" {
				_ = exporter.SendWebhook(webhookURL, []exporter.Turn{turn})
			}

			if delaySec > 0 && i < len(prompts)-1 {
				time.Sleep(time.Duration(delaySec) * time.Second)
			}
		}

		if outputFile != "" && len(turns) > 0 {
			if strings.HasSuffix(strings.ToLower(outputFile), ".json") {
				_ = exporter.SaveJSON(outputFile, turns)
			} else {
				_ = exporter.SaveMarkdown(outputFile, turns)
			}
			fmt.Printf("\n%s\n", green("✓ Saved %d responses to %s", len(turns), outputFile))
		}

		return nil
	},
}

func init() {
	batchCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Path to text file containing one prompt per line")
	batchCmd.Flags().IntVarP(&delaySec, "delay", "d", 3, "Delay between prompts in seconds")
	_ = batchCmd.MarkFlagRequired("input")

	RootCmd.AddCommand(batchCmd)
}