package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"chatgpt-spider/internal/server"
	"chatgpt-spider/internal/spider"
)

var port int

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start an OpenAI-compatible local REST API server",
	Long: `Start a local HTTP server that exposes an OpenAI-compatible /v1/chat/completions API.

You can point Cursor, LangChain, or any application at:
http://localhost:8080/v1

Endpoints:
• POST /v1/chat/completions (supports streaming and non-streaming)
• GET  /v1/models
• GET  /v1/history
• GET  /health`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cyan := color.New(color.FgCyan, color.Bold).SprintfFunc()
		green := color.New(color.FgGreen).SprintfFunc()
		yellow := color.New(color.FgYellow).SprintfFunc()

		fmt.Println(cyan("━━━ Starting chatgpt-spider REST API Server ━━━"))
		fmt.Printf("%s: http://localhost:%d\n", green("Listening on"), port)
		fmt.Println(yellow("Initializing ChatGPT engine in background..."))

		eng, err := spider.NewEngine(spider.Options{
			Headless:     headless,
			Anonymous:    anon,
			SessionToken: sessionToken,
			Debug:        debug,
		})
		if err != nil {
			return fmt.Errorf("failed to launch engine: %w", err)
		}
		defer eng.Close()

		if err := eng.Initialize("", temporaryChat); err != nil {
			return fmt.Errorf("failed to initialize ChatGPT: %w", err)
		}

		fmt.Printf("\n%s\n", green("✓ Engine connected! OpenAI-compatible endpoints ready:"))
		fmt.Printf("  • %s http://localhost:%d/v1/chat/completions\n", cyan("POST"), port)
		fmt.Printf("  • %s  http://localhost:%d/v1/models\n", cyan("GET"), port)
		fmt.Printf("  • %s  http://localhost:%d/v1/history\n", cyan("GET"), port)
		fmt.Printf("  • %s  http://localhost:%d/health\n\n", cyan("GET"), port)

		srv := server.NewServer(eng, port, temporaryChat)
		return srv.Start()
	},
}

func init() {
	serveCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port for the HTTP REST API server")
	RootCmd.AddCommand(serveCmd)
}
