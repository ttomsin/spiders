package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"chatgpt-spider/internal/browser"
	"chatgpt-spider/internal/conversation"
	"chatgpt-spider/internal/exporter"
	"chatgpt-spider/internal/interceptor"
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

		inst, err := browser.Launch(browser.Options{
			Headless:     headless,
			Anonymous:    anon,
			SessionToken: sessionToken,
			Debug:        debug,
		})
		if err != nil {
			return err
		}
		defer inst.Close()

		// Intercept Ctrl+C (SIGINT) to ensure Chromium is gracefully terminated
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Println(dim("\nExiting session..."))
			_ = inst.Close()
			os.Exit(0)
		}()

		page := inst.Page
		itc, err := interceptor.NewInterceptor(page)
		if err != nil {
			return fmt.Errorf("failed to setup interceptor: %w", err)
		}
		defer itc.Stop()

		if chatSessionID != "" {
			fmt.Printf("%s\n", yellow("Connecting and loading conversation %s...", chatSessionID))
			if err := conversation.OpenConversation(page, chatSessionID); err != nil {
				return err
			}
		} else {
			fmt.Println(yellow("Connecting to ChatGPT..."))
			if err := page.Navigate("https://chatgpt.com"); err != nil {
				return err
			}
			_ = page.WaitLoad()
			time.Sleep(3 * time.Second)
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
				if err := conversation.NewChat(page); err != nil {
					fmt.Printf("%s\n", color.YellowString("Notice: could not start new chat: %v", err))
				} else {
					fmt.Println(green("✓ Fresh conversation thread ready."))
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
					if err := conversation.OpenConversation(page, targetID); err != nil {
						fmt.Printf("%s\n", color.RedString("Error switching conversation: %v", err))
					} else {
						fmt.Println(green("✓ Switched to conversation %s", targetID))
					}
				}
				continue
			}

			// Drain any stale buffered response before sending new prompt
			itc.Drain()

			if err := conversation.SendPrompt(page, input); err != nil {
				fmt.Printf("%s\n", color.RedString("Error sending prompt: %v", err))
				continue
			}

			fmt.Print(dim("Waiting for ChatGPT... "))

			var responseText string
			var convID string

			respCh := make(chan string, 1)
			go func() {
				select {
				case resp := <-itc.ResponseChannel():
					select {
					case respCh <- resp:
					default:
					}
				case <-time.After(60 * time.Second):
				}
			}()

			go func() {
				if domText, err := conversation.WaitForCompletion(page, 45*time.Second); err == nil && domText != "" {
					select {
					case respCh <- domText:
					default:
					}
				}
			}()

			select {
			case res := <-respCh:
				responseText = res
				_, convID = itc.GetLastResponse()
				if convID == "" {
					convID = conversation.GetCurrentConversationID(page)
				}
			case <-time.After(60 * time.Second):
				fmt.Printf("%s\n", color.RedString("Response timeout: took longer than 60s"))
				continue
			}

			fmt.Printf("\r%s\n\n", green("ChatGPT >"))
			fmt.Println(responseText)
			if convID != "" {
				fmt.Printf("\n%s\n", dim("[chat-session-id: %s]", convID))
			}

			turns = append(turns, exporter.Turn{
				Prompt:         input,
				Response:       responseText,
				ConversationID: convID,
				Timestamp:      time.Now().UTC().Format(time.RFC3339),
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