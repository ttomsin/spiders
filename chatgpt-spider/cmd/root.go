package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	headless      bool
	debug         bool
	anon          bool
	newChat       bool
	history       bool
	chatSessionID string
	sessionToken  string
	webhookURL    string
	outputFile    string
)

var RootCmd = &cobra.Command{
	Use:   "chatgpt-spider",
	Short: "chatgpt-spider is a high-performance native ChatGPT crawler & automation engine in Go",
	Long: `chatgpt-spider automates ChatGPT interactions through Chrome DevTools Protocol (CDP)
with anti-detection stealth, network response stream interception, and zero external driver dependencies.

Features:
• Native CDP with anti-bot stealth (no chromedriver.exe required)
• Direct Network Hijack of /backend-api/conversation Server-Sent Events (SSE)
• Headless by default (pass --headless=false to view the UI)
• Persistent profile storage (~/.chatgpt-spider/profile) or anonymous guest mode (--anon)
• Interactive terminal REPL, single-prompt execution, and bulk batch processing
• Multi-format export (Markdown, JSON) and direct Webhook data streaming`,
}

func init() {
	RootCmd.PersistentFlags().BoolVar(&headless, "headless", true, "Run browser in headless mode (default true, set --headless=false to view browser)")
	RootCmd.PersistentFlags().BoolVar(&anon, "anon", false, "Run in anonymous / guest mode without using saved credentials or profile")
	RootCmd.PersistentFlags().BoolVar(&newChat, "new-chat", false, "Start a fresh conversation thread before sending prompt")
	RootCmd.PersistentFlags().BoolVar(&history, "history", false, "Fetch and display the message history of the conversation")
	RootCmd.PersistentFlags().StringVar(&chatSessionID, "chat-session-id", "", "Resume a specific existing conversation by ID or URL (e.g., 67c9b2e1-...)")
	RootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Keep browser inspector / devtools open")
	RootCmd.PersistentFlags().StringVarP(&sessionToken, "session-token", "t", "", "ChatGPT __Secure-next-auth.session-token cookie")
	RootCmd.PersistentFlags().StringVar(&webhookURL, "webhook-url", "", "Custom HTTP webhook endpoint to dispatch conversation turns")
	RootCmd.PersistentFlags().StringVarP(&outputFile, "output", "o", "", "Destination output file (.md or .json)")
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(color.RedString("Error: %v", err))
		os.Exit(1)
	}
}