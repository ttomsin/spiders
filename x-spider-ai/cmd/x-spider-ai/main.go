package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"x-spider-ai/internal/browser"
	"x-spider-ai/internal/mcp"
	"x-spider-ai/internal/server"
	"x-spider-ai/pkg/xspiderai"
)

var (
	version   = "dev"
	commit    = "none"
	date      = "unknown"

	headlessFlag bool
	accountFlag  string
	proxyFlag    string
	portFlag     int
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "xsai",
		Aliases: []string{"x-spider-ai"},
		Short:   "xsai: Autonomous Twitter/X action & perception engine for AI agents",
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	}

	rootCmd.PersistentFlags().BoolVar(&headlessFlag, "headless", true, "Run browser in headless mode")
	rootCmd.PersistentFlags().StringVarP(&accountFlag, "account", "a", "", "Target account ID or screen name to use")
	rootCmd.PersistentFlags().StringVar(&proxyFlag, "proxy", "", "Optional HTTP/SOCKS proxy")

	// Start / Init Command
	initCmd := &cobra.Command{
		Use:     "init",
		Aliases: []string{"start", "setup"},
		Short:   "Initialize secure encrypted session vault and environment on this machine",
		RunE: func(cmd *cobra.Command, args []string) error {
			sm := browser.NewSessionManager("")
			if err := sm.InitStorage(); err != nil {
				return fmt.Errorf("failed to initialize vault: %w", err)
			}
			fmt.Printf("✨ xsai initialized successfully!\n")
			fmt.Printf("🔒 Encrypted Vault: %s\n", sm.GetDBPath())
			fmt.Printf("🔑 AES-256 GCM key derived from machine profile.\n")

			accounts, _ := sm.ListAccounts()
			if len(accounts) == 0 {
				fmt.Println("\nNext step: authenticate your account:")
				fmt.Println("  xsai login <auth_token> [ct0] --id main --handle @_your_handle")
				fmt.Println("  or start interactive browser login:")
				fmt.Println("  xsai login")
			} else {
				fmt.Printf("Active accounts: %d\n", len(accounts))
			}

			fmt.Println("\n💡 Tip for Windows users:")
			fmt.Println("  If Windows Smart App Control or Defender flags a newly compiled binary, run:")
			fmt.Println("  Unblock-File .\\xsai.exe")
			return nil
		},
	}

	// MCP Server command
	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start Model Context Protocol (MCP) server over standard I/O",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := xspiderai.NewClient(xspiderai.ClientOptions{
				Headless: headlessFlag,
				ProxyURL: proxyFlag,
			})
			if err != nil {
				return err
			}
			defer client.Close()

			srv := mcp.NewServer(client)
			return srv.ServeStdio()
		},
	}

	// HTTP Server command
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Start HTTP / JSON REST API server for AI tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := xspiderai.NewClient(xspiderai.ClientOptions{
				Headless: headlessFlag,
				ProxyURL: proxyFlag,
			})
			if err != nil {
				return err
			}
			defer client.Close()

			httpSrv := server.NewHTTPServer(client, portFlag)
			return httpSrv.Start()
		},
	}
	serverCmd.Flags().IntVarP(&portFlag, "port", "p", 8080, "Port to listen on")

	// CLI Actions
	var (
		loginAccountID string
		loginHandle    string
	)

	loginCmd := &cobra.Command{
		Use:   "login [auth_token] [ct0]",
		Short: "Log in interactively via browser, or provide auth_token [ct0] directly",
		RunE: func(cmd *cobra.Command, args []string) error {
			targetAcc := loginAccountID
			if targetAcc == "" {
				targetAcc = accountFlag
			}
			if targetAcc == "" {
				targetAcc = "default"
			}

			// If tokens are provided directly
			if len(args) >= 1 {
				authToken := args[0]
				ct0 := ""
				if len(args) > 1 {
					ct0 = args[1]
				}

				client, err := xspiderai.NewClient(xspiderai.ClientOptions{
					AccountID: targetAcc,
					Headless:  headlessFlag,
				})
				if err != nil {
					return err
				}
				defer client.Close()

				if err := client.LoginAccount(targetAcc, loginHandle, authToken, ct0); err != nil {
					return err
				}
				fmt.Printf("Login successful for account %q and session encrypted in SQLite!\n", targetAcc)
				return nil
			}

			// Interactive mode: open visible browser and auto-capture login cookies
			fmt.Printf("[x-spider-ai] No tokens passed. Starting interactive browser login for account %q...\n", targetAcc)
			client, err := xspiderai.NewClient(xspiderai.ClientOptions{
				AccountID: targetAcc,
				Headless:  false, // Must be visible for user interaction
			})
			if err != nil {
				return err
			}
			defer client.Close()

			if err := client.LoginInteractive(180); err != nil {
				return err
			}
			fmt.Printf("Interactive login completed for account %q! You can now use x-spider-ai headlessly.\n", targetAcc)
			return nil
		},
	}
	loginCmd.Flags().StringVar(&loginAccountID, "id", "", "Account identifier (e.g. main, bot1)")
	loginCmd.Flags().StringVar(&loginHandle, "handle", "", "Twitter screen name / handle (e.g. @_ttomsin)")

	var mediaPaths []string
	postCmd := &cobra.Command{
		Use:   "post <text>",
		Short: "Post a tweet directly from CLI (supports --media)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := xspiderai.NewClient(xspiderai.ClientOptions{
				AccountID: accountFlag,
				Headless:  headlessFlag,
			})
			if err != nil {
				return err
			}
			defer client.Close()

			res, err := client.PostTweet(xspiderai.PostTweetOptions{
				Text:           args[0],
				MediaFilePaths: mediaPaths,
			})
			if err != nil {
				return err
			}
			if res.TweetID != "" {
				fmt.Printf("Posted successfully! Tweet ID: %s | URL: %s\n", res.TweetID, res.TweetURL)
			} else {
				fmt.Printf("Posted successfully: %+v\n", res)
			}
			return nil
		},
	}
	postCmd.Flags().StringSliceVarP(&mediaPaths, "media", "m", []string{}, "Paths to images or videos to upload")

	logoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "Clear the saved session from the SQLite database",
		RunE: func(cmd *cobra.Command, args []string) error {
			sm := browser.NewSessionManager("")
			if err := sm.ClearSession(); err != nil {
				return err
			}
			fmt.Println("Session cleared successfully from SQLite database.")
			return nil
		},
	}

	dbCmd := &cobra.Command{
		Use:   "db",
		Short: "Manage the local SQLite session database",
	}

	dbInfoCmd := &cobra.Command{
		Use:   "info",
		Short: "Show SQLite database path and session status",
		RunE: func(cmd *cobra.Command, args []string) error {
			sm := browser.NewSessionManager("")
			fmt.Printf("SQLite DB Path: %s\n", sm.GetDBPath())
			if sess, err := sm.LoadSession(); err == nil {
				fmt.Printf("Status: Active Session Found\nLast Updated: %s\n", sess.UpdatedAt)
			} else {
				fmt.Printf("Status: No Active Session (%v)\n", err)
			}
			return nil
		},
	}

	dbDeleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Permanently delete the SQLite session database file",
		RunE: func(cmd *cobra.Command, args []string) error {
			sm := browser.NewSessionManager("")
			path := sm.GetDBPath()
			if err := sm.DeleteDatabase(); err != nil {
				return err
			}
			fmt.Printf("Deleted SQLite database file: %s\n", path)
			return nil
		},
	}

	likeCmd := &cobra.Command{
		Use:   "like <tweet_id>",
		Short: "Like a tweet by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := xspiderai.NewClient(xspiderai.ClientOptions{
				AccountID: accountFlag,
				Headless:  headlessFlag,
			})
			if err != nil {
				return err
			}
			defer client.Close()

			res, err := client.LikeTweet(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Liked successfully: %+v\n", res)
			return nil
		},
	}

	unlikeCmd := &cobra.Command{
		Use:   "unlike <tweet_id>",
		Short: "Unlike a tweet by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := xspiderai.NewClient(xspiderai.ClientOptions{
				AccountID: accountFlag,
				Headless:  headlessFlag,
			})
			if err != nil {
				return err
			}
			defer client.Close()

			res, err := client.UnlikeTweet(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Unliked successfully: %+v\n", res)
			return nil
		},
	}

	quoteCmd := &cobra.Command{
		Use:   "quote <tweet_id> [comment]",
		Short: "Quote tweet an existing tweet with an optional comment",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			tweetID := args[0]
			comment := ""
			if len(args) > 1 {
				comment = args[1]
			}

			client, err := xspiderai.NewClient(xspiderai.ClientOptions{
				AccountID: accountFlag,
				Headless:  headlessFlag,
			})
			if err != nil {
				return err
			}
			defer client.Close()

			res, err := client.QuoteTweet(tweetID, comment)
			if err != nil {
				return err
			}
			if res.TweetID != "" {
				fmt.Printf("Quoted successfully! Tweet ID: %s | URL: %s\n", res.TweetID, res.TweetURL)
			} else {
				fmt.Printf("Quoted successfully: %+v\n", res)
			}
			return nil
		},
	}

	retweetCmd := &cobra.Command{
		Use:   "retweet <tweet_id>",
		Short: "Retweet a tweet by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := xspiderai.NewClient(xspiderai.ClientOptions{
				AccountID: accountFlag,
				Headless:  headlessFlag,
			})
			if err != nil {
				return err
			}
			defer client.Close()

			res, err := client.Retweet(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Retweeted successfully: %+v\n", res)
			return nil
		},
	}

	bookmarkCmd := &cobra.Command{
		Use:   "bookmark <tweet_id>",
		Short: "Bookmark a tweet by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := xspiderai.NewClient(xspiderai.ClientOptions{
				AccountID: accountFlag,
				Headless:  headlessFlag,
			})
			if err != nil {
				return err
			}
			defer client.Close()

			res, err := client.BookmarkTweet(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Bookmarked successfully: %+v\n", res)
			return nil
		},
	}

	followCmd := &cobra.Command{
		Use:   "follow <screen_name>",
		Short: "Follow a user by handle",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := xspiderai.NewClient(xspiderai.ClientOptions{
				AccountID: accountFlag,
				Headless:  headlessFlag,
			})
			if err != nil {
				return err
			}
			defer client.Close()

			res, err := client.FollowUser(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Followed successfully: %+v\n", res)
			return nil
		},
	}

	unfollowCmd := &cobra.Command{
		Use:   "unfollow <screen_name>",
		Short: "Unfollow a user by handle",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := xspiderai.NewClient(xspiderai.ClientOptions{
				AccountID: accountFlag,
				Headless:  headlessFlag,
			})
			if err != nil {
				return err
			}
			defer client.Close()

			res, err := client.UnfollowUser(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Unfollowed successfully: %+v\n", res)
			return nil
		},
	}

	profileCmd := &cobra.Command{
		Use:   "profile <screen_name>",
		Short: "Get profile information for a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := xspiderai.NewClient(xspiderai.ClientOptions{
				AccountID: accountFlag,
				Headless:  headlessFlag,
			})
			if err != nil {
				return err
			}
			defer client.Close()

			prof, err := client.GetProfile(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Profile for @%s:\n  Name: %s\n  Bio: %s\n  Followers: %d\n  Following: %d\n  Posts: %d\n",
				prof.ScreenName, prof.Name, prof.Description, prof.FollowersCount, prof.FriendsCount, prof.StatusesCount)
			return nil
		},
	}

	meCmd := &cobra.Command{
		Use:   "me",
		Short: "Get the authenticated account's profile details",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := xspiderai.NewClient(xspiderai.ClientOptions{
				AccountID: accountFlag,
				Headless:  headlessFlag,
			})
			if err != nil {
				return err
			}
			defer client.Close()

			prof, err := client.GetMyProfile()
			if err != nil {
				return err
			}
			fmt.Printf("Logged in as @%s:\n  Name: %s\n  Bio: %s\n  Followers: %d\n  Following: %d\n  Posts: %d\n",
				prof.ScreenName, prof.Name, prof.Description, prof.FollowersCount, prof.FriendsCount, prof.StatusesCount)
			return nil
		},
	}

	var (
		discoverQueries        []string
		discoverSince          string
		discoverUntil          string
		discoverMaxResults     int
		discoverSort           string
		discoverMinEngagement  int
		discoverIncludeReplies bool
	)

	discoverCmd := &cobra.Command{
		Use:   "discover [query...]",
		Short: "Multi-angle semantic discovery on X across queries, dates, engagement, and replies",
		RunE: func(cmd *cobra.Command, args []string) error {
			queries := append(discoverQueries, args...)
			if len(queries) == 0 {
				return fmt.Errorf("at least one search query must be provided via arguments or -q/--query")
			}

			client, err := xspiderai.NewClient(xspiderai.ClientOptions{
				Headless: headlessFlag,
			})
			if err != nil {
				return err
			}
			defer client.Close()

			results, err := client.Discover(xspiderai.DiscoverOptions{
				Queries:        queries,
				Since:          discoverSince,
				Until:          discoverUntil,
				MaxResults:     discoverMaxResults,
				Sort:           discoverSort,
				MinEngagement:  discoverMinEngagement,
				IncludeReplies: discoverIncludeReplies,
			})
			if err != nil {
				return err
			}

			fmt.Printf("Discovered %d tweets across %d search queries:\n\n", len(results), len(queries))
			for i, tw := range results {
				fmt.Printf("[%d] ID: %s | @%s (%s)\n    Likes: %d | Retweets: %d | Replies: %d\n    Text: %s\n    URL: %s\n\n",
					i+1, tw.ID, tw.Author.ScreenName, tw.CreatedAt, tw.FavoriteCount, tw.RetweetCount, tw.ReplyCount, tw.FullText, tw.TweetURL)
			}
			return nil
		},
	}

	discoverCmd.Flags().StringSliceVarP(&discoverQueries, "query", "q", []string{}, "Search queries (can be specified multiple times)")
	discoverCmd.Flags().StringVar(&discoverSince, "since", "", "Lower date boundary (YYYY-MM-DD)")
	discoverCmd.Flags().StringVar(&discoverUntil, "until", "", "Upper date boundary (YYYY-MM-DD)")
	discoverCmd.Flags().IntVarP(&discoverMaxResults, "max", "m", 20, "Maximum number of tweets to return")
	discoverCmd.Flags().StringVarP(&discoverSort, "sort", "s", "relevance", "Sort order: relevance, recent, oldest, engagement")
	discoverCmd.Flags().IntVar(&discoverMinEngagement, "min-engagement", 0, "Minimum engagement count (likes + retweets)")
	discoverCmd.Flags().BoolVar(&discoverIncludeReplies, "include-replies", false, "Include replies in search results")

	accountsCmd := &cobra.Command{
		Use:   "accounts",
		Short: "Manage multiple Twitter/X accounts in the local encrypted database",
	}

	accountsListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all saved Twitter/X account credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			sm := browser.NewSessionManager("")
			accounts, err := sm.ListAccounts()
			if err != nil {
				return err
			}
			if len(accounts) == 0 {
				fmt.Println("No accounts registered. Use 'login [auth_token] [ct0] --id <account_id>' to add one.")
				return nil
			}
			fmt.Printf("Registered Accounts (%d):\n", len(accounts))
			for i, a := range accounts {
				activeTag := ""
				if a.IsActive {
					activeTag = " [ACTIVE]"
				}
				handleTag := ""
				if a.ScreenName != "" {
					handleTag = fmt.Sprintf(" (@%s)", a.ScreenName)
				}
				fmt.Printf("[%d] %s%s%s (Updated: %s)\n", i+1, a.ID, handleTag, activeTag, a.UpdatedAt)
			}
			return nil
		},
	}

	accountsSwitchCmd := &cobra.Command{
		Use:   "switch <account_id>",
		Short: "Switch the default active account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sm := browser.NewSessionManager("")
			if err := sm.SwitchAccount(args[0]); err != nil {
				return err
			}
			fmt.Printf("Successfully switched active account to %q\n", args[0])
			return nil
		},
	}

	accountsDeleteCmd := &cobra.Command{
		Use:   "delete <account_id>",
		Short: "Delete a stored account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sm := browser.NewSessionManager("")
			if err := sm.DeleteAccount(args[0]); err != nil {
				return err
			}
			fmt.Printf("Deleted account %q from database\n", args[0])
			return nil
		},
	}

	accountsCmd.AddCommand(accountsListCmd, accountsSwitchCmd, accountsDeleteCmd)

	dbCmd.AddCommand(dbInfoCmd, dbDeleteCmd)

	rootCmd.AddCommand(initCmd, mcpCmd, serverCmd, loginCmd, accountsCmd, postCmd, quoteCmd, likeCmd, unlikeCmd, retweetCmd, bookmarkCmd, followCmd, unfollowCmd, profileCmd, meCmd, discoverCmd, logoutCmd, dbCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
