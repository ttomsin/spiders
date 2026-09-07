package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"chatgpt-spider/internal/exporter"
	"chatgpt-spider/internal/spider"
)

var chatCmd = &cobra.Command{
	Use:   "chat [optional initial prompt]",
	Short: "Start an interactive conversational terminal session with ChatGPT",
	RunE: func(cmd *cobra.Command, args []string) error {
		cyan := color.New(color.FgCyan, color.Bold).SprintfFunc()
		green := color.New(color.FgGreen, color.Bold).SprintfFunc()
		yellow := color.New(color.FgYellow).SprintfFunc()
		dim := color.New(color.FgHiBlack).SprintfFunc()

		fmt.Println(cyan("Starting interactive ChatGPT session..."))
		fmt.Println(dim("Commands: '/new' to start new, '/open <id>' to switch conversation, 'exit' to quit.\n"))

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
			fmt.Printf("%s\n", yellow("Connecting and loading conversation %s...", chatSessionID))
		} else {
			fmt.Println(yellow("Connecting to ChatGPT..."))
		}

		if err := eng.Initialize(chatSessionID); err != nil {
			return err
		}

		var turns []exporter.Turn
		scanner := bufio.NewScanner(os.Stdin)

		initialInput := strings.TrimSpace(strings.Join(args, " "))
		firstTurn := true

		for {
			var input string
			if firstTurn && initialInput != "" {
				input = initialInput
				firstTurn = false
				fmt.Printf("\n%s %s\n", cyan("You >"), input)
			} else {
				firstTurn = false
				fmt.Printf("\n%s ", cyan("You >"))
				if !scanner.Scan() {
					break
				}
				input = strings.TrimSpace(scanner.Text())
			}

			if input == "" {
				continue
			}
			if strings.EqualFold(input, "exit") || strings.EqualFold(input, "quit") {
				break
			}
			if strings.EqualFold(input, "/new") || strings.EqualFold(input, "/clear") {
				fmt.Println(yellow("Starting a fresh conversation thread..."))
				if err := eng.NewChat(); err != nil {
					fmt.Printf("%s\n", color.YellowString("Notice: could not start new chat: %v", err))
				} else {
					fmt.Println(green("✓ Fresh conversation thread ready."))
				}
				continue
			}
			if strings.EqualFold(input, "/history") {
				hist, err := eng.GetHistory()
				if err != nil || len(hist) == 0 {
					fmt.Println(yellow("No visible messages found in this conversation."))
				} else {
					fmt.Println(cyan("━━━ Conversation History ━━━"))
					for _, turn := range hist {
						if turn.Role == "user" {
							fmt.Printf("\n%s: %s\n", cyan("User"), turn.Content)
						} else {
							fmt.Printf("\n%s: %s\n", green("ChatGPT"), turn.Content)
						}
					}
					fmt.Println(cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
				}
				continue
			}
			if strings.EqualFold(input, "/open") || strings.EqualFold(input, "/load") {
				fmt.Println(yellow("Usage: /open <session-id or url>"))
				continue
			}
			if strings.HasPrefix(input, "/open ") || strings.HasPrefix(input, "/load ") {
				parts := strings.SplitN(input, " ", 2)
				if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
					targetID := strings.TrimSpace(parts[1])
					fmt.Printf("%s\n", yellow("Switching to conversation %s...", targetID))
					if err := eng.SwitchConversation(targetID); err != nil {
						fmt.Printf("%s\n", color.RedString("Error switching conversation: %v", err))
					} else {
						fmt.Println(green("✓ Switched to conversation %s", targetID))
					}
				}
				continue
			}

			fmt.Printf("\n%s ", green("ChatGPT >"))

			resp, err := eng.Prompt(cmd.Context(), spider.PromptRequest{
				Prompt:  input,
				Format:  formatSpec,
				OnToken: func(token string) {
					fmt.Print(token)
				},
			})
			if err != nil {
				fmt.Printf("\n%s\n", color.RedString("Error: %v", err))
				continue
			}

			fmt.Println()
			if resp.ConversationID != "" {
				fmt.Printf("%s\n", dim("[chat-session-id: %s]", resp.ConversationID))
			}

			turns = append(turns, exporter.Turn{
				Prompt:         input,
				Response:       resp.Text,
				ConversationID: resp.ConversationID,
				Timestamp:      resp.Timestamp,
			})

			if webhookURL != "" {
				_ = exporter.SendWebhook(webhookURL, turns[len(turns)-1:])
			}
		}

		if outputFile != "" && len(turns) > 0 {
			if strings.HasSuffix(strings.ToLower(outputFile), ".json") {
				_ = exporter.SaveJSON(outputFile, turns)
			} else {
				_ = exporter.SaveMarkdown(outputFile, turns)
			}
			fmt.Printf("\n%s\n", green("✓ Entire conversation saved to %s", outputFile))
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(chatCmd)
}