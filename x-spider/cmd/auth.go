package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"x-spider/internal/auth"
)

// authCmd groups authentication commands
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage saved Twitter authentication credentials and token pool",
	Long: `Manage your Twitter auth_tokens securely stored on your computer.
Once saved, x-spider will automatically load all accounts into a Token Pool
and rotate them across rate-limited sessions without needing to pass flags or update config files.`,
	Example: `  # Set one or more tokens (replaces current pool)
  x-spider auth set-token 1a2b3c4d5e... 9z8y7x6w5v...

  # Add an account to the existing pool
  x-spider auth add-token a1b2c3d4e5...

  # Check stored pool status
  x-spider auth status

  # Remove account #2
  x-spider auth remove 2

  # Clear all stored tokens
  x-spider auth clear`,
}

var setTokenCmd = &cobra.Command{
	Use:   "set-token [tokens...]",
	Short: "Store one or more Twitter auth_tokens securely in your local pool",
	Long: `Encrypts and stores Twitter auth_tokens using AES-256 GCM in your user home directory.
You can pass one or more tokens as arguments:
  x-spider auth set-token <token1> <token2> ...

If no token is provided as an argument, you will be prompted to enter tokens interactively.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var tokens []string

		if len(args) > 0 {
			for _, a := range args {
				trimmed := strings.TrimSpace(a)
				if trimmed != "" {
					tokens = append(tokens, trimmed)
				}
			}
		} else {
			fmt.Println("Enter Twitter auth_tokens (input hidden). Press Enter on an empty prompt when done:")
			for i := 1; ; i++ {
				fmt.Printf("Token #%d: ", i)
				bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Println()
				if err != nil {
					return fmt.Errorf("failed to read token: %w", err)
				}
				trimmed := strings.TrimSpace(string(bytePassword))
				if trimmed == "" {
					break
				}
				if len(trimmed) < 20 {
					fmt.Println(color.YellowString("⚠ Token is shorter than 20 characters; please verify it is a valid auth_token cookie."))
				}
				tokens = append(tokens, trimmed)
			}
		}

		if len(tokens) == 0 {
			return fmt.Errorf("no tokens provided")
		}

		for _, t := range tokens {
			if len(t) < 20 {
				return fmt.Errorf("invalid token '%s': Twitter auth tokens are typically longer than 20 characters", auth.MaskToken(t))
			}
		}

		if err := auth.SaveAuthTokens(tokens); err != nil {
			return fmt.Errorf("failed to save tokens: %w", err)
		}

		green := color.New(color.FgGreen, color.Bold).SprintfFunc()
		fmt.Printf("\n%s\n", green("✓ Successfully stored %d Twitter auth_token(s) in %s", len(tokens), auth.GetStorageFilePath()))
		fmt.Println("x-spider will automatically rotate across these accounts when rate limits occur.")
		return nil
	},
}

var addTokenCmd = &cobra.Command{
	Use:   "add-token [token]",
	Short: "Add an additional Twitter auth_token to the existing pool",
	Long: `Appends a new Twitter auth_token to your existing encrypted pool without overwriting existing accounts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var token string
		if len(args) > 0 {
			token = strings.TrimSpace(args[0])
		} else {
			fmt.Print("Enter additional Twitter auth_token (input hidden): ")
			bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("failed to read token: %w", err)
			}
			token = strings.TrimSpace(string(bytePassword))
		}

		if len(token) < 20 {
			return fmt.Errorf("invalid token: Twitter auth tokens are typically longer than 20 characters")
		}

		if err := auth.AddAuthToken(token); err != nil {
			return fmt.Errorf("failed to add token: %w", err)
		}

		green := color.New(color.FgGreen, color.Bold).SprintfFunc()
		fmt.Printf("\n%s\n", green("✓ Added new Twitter account (%s) to the token pool.", auth.MaskToken(token)))
		return nil
	},
}

var removeTokenCmd = &cobra.Command{
	Use:     "remove <index>",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove a specific token from the pool by index",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var idx int
		_, err := fmt.Sscanf(args[0], "%d", &idx)
		if err != nil || idx < 1 {
			return fmt.Errorf("invalid index: please provide a positive integer (e.g. 'x-spider auth remove 1')")
		}

		if err := auth.RemoveAuthToken(idx); err != nil {
			return err
		}

		yellow := color.New(color.FgYellow).SprintfFunc()
		fmt.Printf("%s\n", yellow("✓ Account #%d removed from token pool.", idx))
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:     "status",
	Aliases: []string{"list", "ls"},
	Short:   "Check the status and accounts in the stored Twitter token pool",
	RunE: func(cmd *cobra.Command, args []string) error {
		cyan := color.New(color.FgCyan).SprintfFunc()
		green := color.New(color.FgGreen).SprintfFunc()
		yellow := color.New(color.FgYellow).SprintfFunc()

		tokens, err := auth.GetAuthTokens()
		if err != nil {
			return fmt.Errorf("error reading stored credentials: %w", err)
		}

		fmt.Printf("%s: %s\n", cyan("Credentials Storage"), auth.GetStorageFilePath())
		fmt.Printf("%s: %s\n", cyan("Encryption"), "AES-256 GCM (Machine-derived key)")

		if len(tokens) == 0 {
			fmt.Printf("%s: %s\n\n", cyan("Status"), yellow("No auth_token saved."))
			fmt.Println("Run 'x-spider auth set-token' or 'x-spider auth add-token' to save credentials.")
		} else {
			fmt.Printf("%s: %s (%d account(s) ready)\n\n", cyan("Status"), green("Active / Stored"), len(tokens))
			fmt.Println("Saved Accounts in Pool:")
			for i, tok := range tokens {
				fmt.Printf("  [%d] %s\n", i+1, auth.MaskToken(tok))
			}
			fmt.Println()
			fmt.Println("x-spider will automatically rotate across these accounts during crawling.")
		}
		return nil
	},
}

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove all stored Twitter credentials from your computer",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := auth.ClearAuthToken(); err != nil {
			return fmt.Errorf("failed to clear credentials: %w", err)
		}
		yellow := color.New(color.FgYellow).SprintfFunc()
		fmt.Printf("%s\n", yellow("✓ All saved Twitter credentials removed."))
		return nil
	},
}

func init() {
	authCmd.AddCommand(setTokenCmd)
	authCmd.AddCommand(addTokenCmd)
	authCmd.AddCommand(removeTokenCmd)
	authCmd.AddCommand(statusCmd)
	authCmd.AddCommand(clearCmd)

	RootCmd.AddCommand(authCmd)
}
