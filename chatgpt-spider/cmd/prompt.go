package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"chatgpt-spider/internal/browser"
	"chatgpt-spider/internal/conversation"
	"chatgpt-spider/internal/exporter"
	"chatgpt-spider/internal/interceptor"
)

var promptCmd = &cobra.Command{
	Use:   "prompt <text>",
	Short: "Send a single prompt to ChatGPT and retrieve the response",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		promptText := strings.Join(args, " ")

		cyan := color.New(color.FgCyan).SprintfFunc()
		green := color.New(color.FgGreen).SprintfFunc()
		yellow := color.New(color.FgYellow).SprintfFunc()

		fmt.Printf("%s: %s\n", cyan("Prompt"), promptText)
		fmt.Printf("%s\n", yellow("Launching browser..."))

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

		page := inst.Page
		itc, err := interceptor.NewInterceptor(page)
		if err != nil {
			return fmt.Errorf("failed to setup interceptor: %w", err)
		}
		defer itc.Stop()

		if chatSessionID != "" {
			fmt.Printf("%s\n", yellow("Resuming conversation: %s...", chatSessionID))
			if err := conversation.OpenConversation(page, chatSessionID); err != nil {
				return err
			}
		} else {
			fmt.Printf("%s\n", yellow("Navigating to ChatGPT..."))
			if err := page.Navigate("https://chatgpt.com"); err != nil {
				return fmt.Errorf("navigation error: %w", err)
			}
			_ = page.WaitLoad()
			time.Sleep(3 * time.Second)

			if newChat {
				fmt.Printf("%s\n", yellow("Starting new chat thread..."))
				if err := conversation.NewChat(page); err != nil {
					fmt.Printf("%s\n", color.YellowString("Notice: new chat trigger error: %v", err))
				}
			}
		}

		fmt.Printf("%s\n", yellow("Sending prompt..."))
		if err := conversation.SendPrompt(page, promptText); err != nil {
			return err
		}

		fmt.Printf("%s\n\n", cyan("Waiting for response..."))

		var responseText string
		var convID string

		// Race SSE interceptor and live DOM watcher so we return the instant it finishes
		respCh := make(chan string, 1)

		// Goroutine 1: Network interceptor
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

		// Goroutine 2: Active DOM completion watcher
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
			return fmt.Errorf("response timeout: ChatGPT took longer than 60s to finish responding")
		}

		fmt.Printf("%s\n\n", green("━━━ ChatGPT Response ━━━"))
		fmt.Println(responseText)
		fmt.Printf("\n%s\n", green("━━━━━━━━━━━━━━━━━━━━━━━━"))

		turns := []exporter.Turn{
			{
				Prompt:         promptText,
				Response:       responseText,
				ConversationID: convID,
				Timestamp:      time.Now().UTC().Format(time.RFC3339),
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