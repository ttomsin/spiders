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

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive conversational terminal session with ChatGPT",
	RunE: func(cmd *cobra.Command, args []string) error {
		cyan := color.New(color.FgCyan, color.Bold).SprintfFunc()
		green := color.New(color.FgGreen, color.Bold).SprintfFunc()
		yellow := color.New(color.FgYellow).SprintfFunc()
		dim := color.New(color.FgHiBlack).SprintfFunc()

		fmt.Println(cyan("Starting interactive ChatGPT session..."))
		fmt.Println(dim("Type 'exit', 'quit', or press Ctrl+C to conclude.\n"))

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

		fmt.Println(yellow("Connecting to ChatGPT..."))
		if err := page.Navigate("https://chatgpt.com"); err != nil {
			return err
		}
		_ = page.WaitLoad()
		time.Sleep(3 * time.Second)

		var turns []exporter.Turn
		scanner := bufio.NewScanner(os.Stdin)

		for {
			fmt.Printf("\n%s ", cyan("You >"))
			if !scanner.Scan() {
				break
			}
			input := strings.TrimSpace(scanner.Text())
			if input == "" {
				continue
			}
			if strings.EqualFold(input, "exit") || strings.EqualFold(input, "quit") {
				break
			}

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
			case <-time.After(60 * time.Second):
				fmt.Printf("%s\n", color.RedString("Response timeout: took longer than 60s"))
				continue
			}

			fmt.Printf("\r%s\n\n", green("ChatGPT >"))
			fmt.Println(responseText)

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