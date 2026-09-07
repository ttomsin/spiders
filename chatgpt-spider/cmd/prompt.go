package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"chatgpt-spider/internal/exporter"
	"chatgpt-spider/internal/spider"
)

var promptCmd = &cobra.Command{
	Use:   "prompt [text]",
	Short: "Send a single prompt to ChatGPT and retrieve the response (or fetch history with --history)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 && !history {
			return fmt.Errorf("prompt text is required (or pass --history to inspect a thread)")
		}

		promptText := strings.Join(args, " ")

		cyan := color.New(color.FgCyan).SprintfFunc()
		green := color.New(color.FgGreen).SprintfFunc()
		yellow := color.New(color.FgYellow).SprintfFunc()

		if promptText != "" {
			fmt.Printf("%s: %s\n", cyan("Prompt"), promptText)
		}
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
		} else {
			fmt.Printf("%s\n", yellow("Navigating to ChatGPT..."))
		}

		if err := eng.Initialize(chatSessionID); err != nil {
			return err
		}

		if history {
			hist, err := eng.GetHistory()
			if err != nil {
				return err
			}
			fmt.Printf("\n%s\n", cyan("━━━ Conversation History ━━━"))
			for _, turn := range hist {
				if turn.Role == "user" {
					fmt.Printf("\n%s: %s\n", cyan("User"), turn.Content)
				} else {
					fmt.Printf("\n%s: %s\n", green("ChatGPT"), turn.Content)
				}
			}
			fmt.Printf("\n%s\n", cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))

			if promptText == "" {
				return nil
			}
		}

		if newChat {
			fmt.Printf("%s\n", yellow("Starting new chat thread..."))
			_ = eng.NewChat()
		}

		fmt.Printf("%s\n\n", green("━━━ ChatGPT Response ━━━"))

		resp, err := eng.Prompt(cmd.Context(), spider.PromptRequest{
			Prompt: promptText,
			OnToken: func(token string) {
				fmt.Print(token)
			},
		})
		if err != nil {
			return err
		}

		fmt.Printf("\n\n%s\n", green("━━━━━━━━━━━━━━━━━━━━━━━━"))
		if resp.ConversationID != "" {
			dim := color.New(color.FgHiBlack).SprintfFunc()
			fmt.Printf("%s\n", dim("Session ID: %s (use --chat-session-id %s to continue)", resp.ConversationID, resp.ConversationID))
		}

		turns := []exporter.Turn{
			{
				Prompt:         promptText,
				Response:       resp.Text,
				ConversationID: resp.ConversationID,
				Timestamp:      resp.Timestamp,
			},
		}

		if outputFile != "" {
			if strings.HasSuffix(strings.ToLower(outputFile), ".json") {
				_ = exporter.SaveJSON(outputFile, turns)
			} else {
				_ = exporter.SaveMarkdown(outputFile, turns)
			}
			fmt.Printf("\n%s: %s\n", cyan("Saved response to"), outputFile)
		}

		if webhookURL != "" {
			_ = exporter.SendWebhook(webhookURL, turns)
			fmt.Printf("%s: %s\n", cyan("Dispatched response to webhook"), webhookURL)
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(promptCmd)
}