package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"x-spider/internal/auth"
	"x-spider/internal/config"
	"x-spider/internal/crawler"
	"x-spider/internal/prompt"
)

const AppVersion = "1.0.0"

var (
	configFilePath string
	cfg            = config.NewDefaultConfig()
)

// ShowWelcomeMessage displays the ASCII spider art and welcome banner
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
		cyan("      / _ \\     "), boldGreen(" __  __           ____        _     _           "),
		cyan("    \\_\\(_)/_/   "), boldGreen(" \\ \\/ /          / ___| _ __ (_) __| | ___ _ __ "),
		cyan("     _//o\\\\_    "), boldGreen("  \\  /  _____   \\___ \\| '_ \\| |/ _` |/ _ \\ '__|"),
		cyan("      /   \\     "), boldGreen("  /  \\ |_____|   ___) | |_) | | (_| |  __/ |   "),
		cyan("     /     \\    "), boldGreen(" /_/\\_\\         |____/| .__/|_|\\__,_|\\___|_|   "),
		cyan("                "), boldGreen("                      |_|                      "),
		"", dim("       [Advanced Twitter/X Crawler • v"+AppVersion+"]"),
	)

	fmt.Print(asciiArt)
	fmt.Printf("\n%s %s\n", boldBlue("x-spider:"), "High-performance automated Twitter/X intelligence collector.")
	fmt.Println(dim("• Anti-detection stealth • Bandwidth-saving media filter • Direct GraphQL capture"))
	fmt.Println(dim("• Safe local execution • Persistent encrypted authentication"))
	fmt.Println()
}

// RootCmd is the primary CLI command
var RootCmd = &cobra.Command{
	Use:   "x-spider",
	Short: "x-spider is a high-performance Twitter/X crawler in Go",
	Long: `x-spider is a high-performance Twitter/X crawler built in Go.
It automates searches, keyword filters, and discussion threads using native DevTools
Protocol browser automation with anti-bot stealth and network response capture.

Results can be exported directly to CSV or Excel (.xlsx) with standardized fields.
Authentication can be saved once on your machine so you don't need to pass tokens repeatedly.`,
	Example: `  # 1. First-time setup: securely store your Twitter auth_token
  x-spider auth set-token

  # 2. Check auth status
  x-spider auth status

  # 3. Simple keyword crawl using saved credentials
  x-spider -s "golang" -l 25

  # 4. Filter by date range and export to Excel (.xlsx)
  x-spider -s "machine learning" -f "01-01-2026" --to "01-02-2026" -l 100 -e xlsx

  # 5. Scrape a specific tweet thread discussion
  x-spider --thread "https://x.com/username/status/1234567890" -l 50

  # 6. Run fully automated from a YAML configuration file
  x-spider -c config.yaml

  # 7. Convert crawled replies into a Gephi network graph edge list
  x-spider gephi -i ./tweets-data/tweets.csv -o ./tweets-data/network_edges.csv`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ShowWelcomeMessage()

		// 1. Load config file if specified or default exists
		if configFilePath != "" {
			fileCfg, err := config.LoadConfigFile(configFilePath)
			if err != nil {
				return fmt.Errorf("error loading config file: %w", err)
			}
			mergeConfig(fileCfg, cfg)
		} else if _, err := os.Stat("config.yaml"); err == nil {
			fileCfg, err := config.LoadConfigFile("config.yaml")
			if err == nil {
				fmt.Println(color.CyanString("ℹ Loaded configuration from config.yaml"))
				mergeConfig(fileCfg, cfg)
			}
		}

		// 2. Apply environment variable overrides
		cfg.ApplyEnvOverrides()

		// 3. Check persistent credential store if auth_tokens are not explicitly configured
		if len(cfg.AuthTokens) == 0 && cfg.AuthToken == "" {
			if storedTokens, err := auth.GetAuthTokens(); err == nil && len(storedTokens) > 0 {
				cfg.AuthTokens = storedTokens
				cfg.AuthToken = storedTokens[0]
				if len(storedTokens) == 1 {
					fmt.Printf("%s\n", color.HiBlackString("ℹ Using saved Twitter credentials (%s)", auth.MaskToken(storedTokens[0])))
				} else {
					fmt.Printf("%s\n", color.HiBlackString("ℹ Using %d saved Twitter accounts in rotation pool (Primary: %s)", len(storedTokens), auth.MaskToken(storedTokens[0])))
				}
			}
		} else if len(cfg.AuthTokens) == 0 && cfg.AuthToken != "" {
			// Single token provided via flag or env
			cfg.AuthTokens = []string{cfg.AuthToken}
		}

		// 4. Prompt interactively for missing credentials if in terminal
		if err := prompt.AskQuestionsPrompts(cfg); err != nil {
			return err
		}

		// 5. Setup context cancellation on interrupt (Ctrl+C)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
			cancel()
		}()

		// 6. Run crawler
		c := crawler.NewCrawler(cfg)
		return c.Run(ctx)
	},
}

// mergeConfig copies values from source into target when target was not explicitly customized
func mergeConfig(source *config.Config, target *config.Config) {
	if target.AuthToken == "" {
		target.AuthToken = source.AuthToken
	}
	if target.SearchKeyword == "" {
		target.SearchKeyword = source.SearchKeyword
	}
	if target.ThreadURL == "" {
		target.ThreadURL = source.ThreadURL
	}
	if target.FromDate == "" {
		target.FromDate = source.FromDate
	}
	if target.ToDate == "" {
		target.ToDate = source.ToDate
	}
	if target.OutputFilename == "" {
		target.OutputFilename = source.OutputFilename
	}
	if target.Proxy == "" {
		target.Proxy = source.Proxy
	}
	if len(source.BlockedExtensions) > 0 {
		target.BlockedExtensions = source.BlockedExtensions
	}
	if source.Limit > 0 && target.Limit == config.DefaultLimit {
		target.Limit = source.Limit
	}
	if source.DelaySeconds > 0 && target.DelaySeconds == config.DefaultDelaySeconds {
		target.DelaySeconds = source.DelaySeconds
	}
	if source.SearchTab != "" {
		target.SearchTab = source.SearchTab
	}
	if source.ExportFormat != "" {
		target.ExportFormat = source.ExportFormat
	}
	if source.CSVInsertMode != "" {
		target.CSVInsertMode = source.CSVInsertMode
	}
	if source.FolderDestination != "" {
		target.FolderDestination = source.FolderDestination
	}
	if !source.Headless {
		target.Headless = false
	}
	if source.Debug {
		target.Debug = true
	}
	if source.EnableExponentialBackoff {
		target.EnableExponentialBackoff = true
	}
	if len(source.AuthTokens) > 0 {
		target.AuthTokens = source.AuthTokens
	}
	if len(source.Proxies) > 0 {
		target.Proxies = source.Proxies
	}
	if source.Chunk != "" && target.Chunk == "" {
		target.Chunk = source.Chunk
	}
	if source.Resume {
		target.Resume = true
	}
	if source.StripURLs {
		target.StripURLs = true
	}
	if source.StripMentions {
		target.StripMentions = true
	}
	if source.StripEmojis {
		target.StripEmojis = true
	}
	if source.MinLength > 0 && target.MinLength == 0 {
		target.MinLength = source.MinLength
	}
	if source.Notifications.DiscordWebhookURL != "" || source.Notifications.TelegramBotToken != "" || source.Notifications.WebhookURL != "" {
		target.Notifications = source.Notifications
	}
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&configFilePath, "config", "c", "", "Path to YAML configuration file")

	RootCmd.Flags().StringVarP(&cfg.AuthToken, "token", "t", "", "Twitter auth_token cookie value (optional if saved via 'x-spider auth set-token')")
	RootCmd.Flags().StringVarP(&cfg.SearchKeyword, "search-keyword", "s", "", "Search query keywords or hashtags")
	RootCmd.Flags().StringVar(&cfg.ThreadURL, "thread", "", "Tweet thread URL (detail crawl mode)")
	RootCmd.Flags().StringVarP(&cfg.FromDate, "from", "f", "", "Start date filter in DD-MM-YYYY format")
	RootCmd.Flags().StringVar(&cfg.ToDate, "to", "", "End date filter in DD-MM-YYYY format")
	RootCmd.Flags().StringVar(&cfg.Chunk, "chunk", "", "Automated date chunking: 'monthly', 'weekly', or 'daily'")
	RootCmd.Flags().BoolVar(&cfg.Resume, "resume", false, "Resume crawl from saved checkpoint file")
	RootCmd.Flags().IntVarP(&cfg.Limit, "limit", "l", config.DefaultLimit, "Maximum number of tweets to collect")
	RootCmd.Flags().IntVarP(&cfg.DelaySeconds, "delay", "d", config.DefaultDelaySeconds, "Delay between tweet requests in seconds")
	RootCmd.Flags().StringVarP(&cfg.OutputFilename, "output-filename", "o", "", "Custom output filename (without extension)")
	RootCmd.Flags().StringVar(&cfg.SearchTab, "tab", config.DefaultSearchTab, "Search tab: 'TOP' for top tweets or 'LATEST' for live timeline")
	RootCmd.Flags().StringVarP(&cfg.ExportFormat, "export-format", "e", config.DefaultExportFormat, "Export file format: 'csv', 'xlsx', 'json', 'jsonl', or 'sqlite'")
	RootCmd.Flags().BoolVar(&cfg.Headless, "headless", true, "Run Chromium in headless mode")
	RootCmd.Flags().BoolVar(&cfg.Debug, "debug", false, "Debug mode (keeps browser open upon errors)")
	RootCmd.Flags().BoolVar(&cfg.EnableExponentialBackoff, "backoff", false, "Enable exponential backoff delays on Twitter rate limits")
	RootCmd.Flags().BoolVar(&cfg.StripURLs, "strip-urls", false, "Remove hyperlinks from tweet text")
	RootCmd.Flags().BoolVar(&cfg.StripMentions, "strip-mentions", false, "Remove @mentions from tweet text")
	RootCmd.Flags().BoolVar(&cfg.StripEmojis, "strip-emojis", false, "Remove Unicode emojis from tweet text")
	RootCmd.Flags().IntVar(&cfg.MinLength, "min-length", 0, "Discard tweets shorter than minimum character length")
}
