package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"chatgpt-spider/internal/browser"
	"chatgpt-spider/internal/conversation"
	"chatgpt-spider/internal/exporter"
	"chatgpt-spider/internal/interceptor"
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

		inst, err := browser.Launch(browser.Options{
			Headless:     headless,
			SessionToken: sessionToken,
			Debug:        debug,
		})
		if err != nil {
			return err
		}
		defer inst.Close()

		page := inst.Page
		itc, err := interceptor.NewInterceptor(page)
		if err != nil {
			return fmt.Errorf("failed to setup interceptor: %w", err)
		}
		defer itc.Stop()

		if err := page.Navigate("https://chatgpt.com"); err != nil {
			return err
		}
		_ = page.WaitLoad()
		time.Sleep(3 * time.Second)

		var turns []exporter.Turn

		for i, promptText := range prompts {
			fmt.Printf("\n%s\n", cyan("━━━ Processing [%d/%d]: %s ━━━", i+1, len(prompts), promptText))

			if err := conversation.SendPrompt(page, promptText); err != nil {
				fmt.Printf("%s\n", color.RedString("Error sending prompt: %v", err))
				continue
			}

			var responseText string
			var convID string

			select {
			case resp := <-itc.ResponseChannel():
				responseText = resp
				_, convID = itc.GetLastResponse()
			case <-time.After(60 * time.Second):
				domText, domErr := conversation.WaitForCompletion(page, 30*time.Second)
				if domErr != nil {
					fmt.Printf("%s\n", color.RedString("Timeout waiting for response: %v", domErr))
					continue
				}
				responseText = domText
			}

			fmt.Printf("%s\n", green("Response captured (%d characters)", len(responseText)))

			turn := exporter.Turn{
				Prompt:         promptText,
				Response:       responseText,
				ConversationID: convID,
				Timestamp:      time.Now().UTC().Format(time.RFC3339),
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