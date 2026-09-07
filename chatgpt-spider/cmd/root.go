package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	headless     bool
	debug        bool
	sessionToken string
	webhookURL   string
	outputFile   string
)

var RootCmd = &cobra.Command{
	Use:   "chatgpt-spider",
	Short: "chatgpt-spider is a high-performance native ChatGPT crawler & automation engine in Go",
	Long: `chatgpt-spider automates ChatGPT interactions through Chrome DevTools Protocol (CDP)
with anti-detection stealth, network response stream interception, and zero external driver dependencies.

Features:
• Native CDP with anti-bot stealth (no chromedriver.exe required)
• Direct Network Hijack of /backend-api/conversation Server-Sent Events (SSE)
• Persistent profile storage (~/.chatgpt-spider/profile) — log in once, stay logged in
• Interactive terminal REPL, single-prompt execution, and bulk batch processing
• Multi-format export (Markdown, JSON) and direct Webhook data streaming`,
}

func init() {
	RootCmd.PersistentFlags().BoolVar(&headless, "headless", false, "Run browser in headless mode (default false to allow initial verification)")
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