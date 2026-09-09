package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"chatgpt-spider/internal/exporter"
	"chatgpt-spider/internal/spider"
)

var promptFile string

var promptCmd = &cobra.Command{
	Use:   "prompt [text]",
	Short: "Send a single prompt to ChatGPT and retrieve the response (or fetch history with --history)",
	RunE: func(cmd *cobra.Command, args []string) error {
		promptText := strings.Join(args, " ")
		var fileContent string

		if promptFile != "" {
			fileBytes, err := os.ReadFile(promptFile)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", promptFile, err)
			}
			fileContent = strings.TrimSpace(string(fileBytes))
		}

		if len(strings.TrimSpace(promptText)) == 0 && len(fileContent) == 0 && !history {
			return fmt.Errorf("prompt text or --file is required (or pass --history to inspect a thread)")
		}

		cyan := color.New(color.FgCyan).SprintfFunc()
		green := color.New(color.FgGreen).SprintfFunc()
		yellow := color.New(color.FgYellow).SprintfFunc()

		if promptText != "" {
			displayPrompt := promptText
			if len(displayPrompt) > 120 {
				displayPrompt = displayPrompt[:120] + "... [truncated]"
			}
			fmt.Printf("%s: %s\n", cyan("Prompt"), displayPrompt)
		}
		if fileContent != "" {
			fmt.Printf("%s: %s (%d bytes)\n", cyan("Attachment"), promptFile, len(fileContent))
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

		if err := eng.Initialize(chatSessionID, temporaryChat); err != nil {
			return fmt.Errorf("failed to initialize ChatGPT: %w", err)
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


		fmt.Printf("%s\n\n", green("━━━ ChatGPT Response ━━━"))

		resp, err := eng.Prompt(cmd.Context(), spider.PromptRequest{
			Prompt:         promptText,
			FileContent:    fileContent,
			Format:         formatSpec,
			NewChat:        newChat || (chatSessionID == ""),
			ConversationID: chatSessionID,
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
			lower := strings.ToLower(outputFile)
			// If user requested a specific format (e.g. json, csv) or file is csv/jsonl/xml/txt, save raw content
			if formatSpec != "" || strings.HasSuffix(lower, ".csv") || strings.HasSuffix(lower, ".jsonl") || strings.HasSuffix(lower, ".xml") || strings.HasSuffix(lower, ".txt") {
				_ = exporter.SaveRaw(outputFile, resp.Text)
			} else if strings.HasSuffix(lower, ".json") {
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

		if sessionDelete && resp.ConversationID != "" {
			fmt.Printf("%s\n", yellow("Deleting session %s from ChatGPT account...", resp.ConversationID))
			if delErr := eng.DeleteConversation(resp.ConversationID); delErr != nil {
				fmt.Printf("%s: %v\n", color.RedString("Failed to delete session"), delErr)
			} else {
				fmt.Println(color.GreenString("✓ Session %s successfully deleted.", resp.ConversationID))
			}
		}

		return nil
	},
}

func init() {
	promptCmd.Flags().StringVarP(&promptFile, "file", "F", "", "Path to a file whose contents should be sent as part of the prompt")
	RootCmd.AddCommand(promptCmd)
}