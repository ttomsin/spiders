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
	formatSpec    string
	sessionDelete bool
)

const AppVersion = "1.0.0"

// ShowWelcomeMessage displays the custom ASCII banner for chatgpt-spider
func ShowWelcomeMessage() {
	cyan := color.New(color.FgHiCyan).SprintFunc()
	boldGreen := color.New(color.FgGreen, color.Bold).SprintFunc()
	dim := color.New(color.FgHiBlack).SprintFunc()
	boldBlue := color.New(color.FgHiBlue, color.Bold).SprintFunc()

	asciiArt := fmt.Sprintf(`
%s     %s
%s     %s
%s      %s
%s       %s
%s      %s
%s                   %s
%s                                        %s
`,
		cyan("      / _ \\     "), boldGreen("        _           _              _     ____        _     _           "),
		cyan("    \\_\\(_)/_/   "), boldGreen("   ____| |__   __ _| |_ __ _ _ __ | |_   / ___| _ __ (_) __| | ___ _ __ "),
		cyan("     _//o\\\\_    "), boldGreen("  / __/| '_ \\ / _` | __/ _` | '_ \\| __|  \\___ \\| '_ \\| |/ _` |/ _ \\ '__|"),
		cyan("      /   \\     "), boldGreen(" | (__ | | | | (_| | || (_| | |_) | |_    ___) | |_) | | (_| |  __/ |   "),
		cyan("     /     \\    "), boldGreen("  \\___|_| |_|\\__,_|\\__\\__, | .__/ \\__|  |____/| .__/|_|\\__,_|\\___|_|   "),
		cyan("                "), boldGreen("                       |___/|_|                |_|                      "),
		"", dim("       [Native ChatGPT Automation Engine & REST Server • v"+AppVersion+"]"),
	)

	fmt.Print(asciiArt)
	fmt.Printf("\n%s %s\n", boldBlue("chatgpt-spider:"), "High-performance native ChatGPT crawler & automation engine in Go.")
	fmt.Println(dim("• Native CDP • Real-time Token Streaming • Zero External Drivers • OpenAI REST Server"))
	fmt.Println()
}

var RootCmd = &cobra.Command{
	Use:   "chatgpt-spider",
	Short: "chatgpt-spider is a high-performance native ChatGPT crawler & automation engine in Go",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Show banner for interactive commands or when help is requested
		if cmd.Name() == "chat" || cmd.Name() == "serve" || cmd.Name() == "chatgpt-spider" {
			ShowWelcomeMessage()
		}
	},
	Long: `chatgpt-spider automates ChatGPT interactions through Chrome DevTools Protocol (CDP)
with anti-detection stealth, network response stream interception, and zero external driver dependencies.

Features:
• Native CDP with anti-bot stealth (no chromedriver.exe required)
• Real-time token streaming with live typewriter effect
• Direct Network Hijack of /backend-api/conversation Server-Sent Events (SSE)
• OpenAI-compatible REST API server (chatgpt-spider serve)
• In-conversation history extraction (--history or /history)
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
	RootCmd.PersistentFlags().StringVarP(&formatSpec, "format", "f", "", "Enforce response format (e.g. json, csv, xml, or custom schema)")
	RootCmd.PersistentFlags().BoolVar(&sessionDelete, "session-delete", false, "Automatically delete the conversation thread from ChatGPT account upon completion")
}

func Execute() {
	if len(os.Args) == 1 || (len(os.Args) == 2 && (os.Args[1] == "--help" || os.Args[1] == "-h" || os.Args[1] == "help")) {
		ShowWelcomeMessage()
	}
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(color.RedString("Error: %v", err))
		os.Exit(1)
	}
}