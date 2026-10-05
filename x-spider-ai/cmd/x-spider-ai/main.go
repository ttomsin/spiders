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
	headlessFlag bool
	proxyFlag    string
	portFlag     int
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "x-spider-ai",
		Short: "x-spider-ai: Pluggable Twitter/X action engine for AI agents",
	}

	rootCmd.PersistentFlags().BoolVar(&headlessFlag, "headless", true, "Run browser in headless mode")
	rootCmd.PersistentFlags().StringVar(&proxyFlag, "proxy", "", "Optional HTTP/SOCKS proxy")

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
	loginCmd := &cobra.Command{
		Use:   "login [auth_token] [ct0]",
		Short: "Log in interactively via browser, or provide auth_token [ct0] directly",
		RunE: func(cmd *cobra.Command, args []string) error {
			// If tokens are provided directly
			if len(args) >= 1 {
				authToken := args[0]
				ct0 := ""
				if len(args) > 1 {
					ct0 = args[1]
				}

				client, err := xspiderai.NewClient(xspiderai.ClientOptions{
					Headless: headlessFlag,
				})
				if err != nil {
					return err
				}
				defer client.Close()

				if err := client.Login(authToken, ct0); err != nil {
					return err
				}
				fmt.Println("Login successful and session encrypted!")
				return nil
			}

			// Interactive mode: open visible browser and auto-capture login cookies
			fmt.Println("[x-spider-ai] No tokens passed. Starting interactive browser login...")
			client, err := xspiderai.NewClient(xspiderai.ClientOptions{
				Headless: false, // Must be visible for user interaction
			})
			if err != nil {
				return err
			}
			defer client.Close()

			if err := client.LoginInteractive(180); err != nil {
				return err
			}
			fmt.Println("Interactive login completed! You can now use x-spider-ai headlessly.")
			return nil
		},
	}

	var mediaPaths []string
	postCmd := &cobra.Command{
		Use:   "post <text>",
		Short: "Post a tweet directly from CLI (supports --media)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := xspiderai.NewClient(xspiderai.ClientOptions{
				Headless: headlessFlag,
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
				Headless: headlessFlag,
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
				Headless: headlessFlag,
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
				Headless: headlessFlag,
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
				Headless: headlessFlag,
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
				Headless: headlessFlag,
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
				Headless: headlessFlag,
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
				Headless: headlessFlag,
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
				Headless: headlessFlag,
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
				Headless: headlessFlag,
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

	dbCmd.AddCommand(dbInfoCmd, dbDeleteCmd)

	rootCmd.AddCommand(mcpCmd, serverCmd, loginCmd, postCmd, quoteCmd, likeCmd, unlikeCmd, retweetCmd, bookmarkCmd, followCmd, unfollowCmd, profileCmd, meCmd, logoutCmd, dbCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
