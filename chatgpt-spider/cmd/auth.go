package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"chatgpt-spider/internal/auth"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage saved ChatGPT authentication credentials",
}

var setTokenCmd = &cobra.Command{
	Use:   "set-token [token]",
	Short: "Store your __Secure-next-auth.session-token securely on your machine",
	RunE: func(cmd *cobra.Command, args []string) error {
		var token string
		if len(args) > 0 {
			token = strings.TrimSpace(args[0])
		} else {
			fmt.Print("Enter ChatGPT __Secure-next-auth.session-token (input hidden): ")
			pass, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Println()
			if err != nil {
				return err
			}
			token = strings.TrimSpace(string(pass))
		}

		if len(token) < 20 {
			return fmt.Errorf("invalid token: session tokens are typically longer than 20 characters")
		}

		if err := auth.SaveSessionToken(token); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}

		fmt.Println(color.GreenString("✓ ChatGPT session token securely saved!"))
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check stored authentication status and profile directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		cyan := color.New(color.FgCyan).SprintfFunc()
		green := color.New(color.FgGreen).SprintfFunc()
		yellow := color.New(color.FgYellow).SprintfFunc()

		profileDir, _ := auth.GetProfileDir()
		token, err := auth.GetSessionToken()
		if err != nil {
			return err
		}

		fmt.Printf("%s: %s\n", cyan("Profile Storage"), profileDir)
		if token == "" {
			fmt.Printf("%s: %s\n", cyan("Session Token"), yellow("None saved (using browser profile cookies)"))
		} else {
			fmt.Printf("%s: %s (%s)\n", cyan("Session Token"), green("Stored"), auth.MaskToken(token))
		}
		return nil
	},
}

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear stored credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := auth.ClearCredentials(); err != nil {
			return err
		}
		fmt.Println(color.YellowString("✓ Stored credentials cleared."))
		return nil
	},
}

func init() {
	authCmd.AddCommand(setTokenCmd)
	authCmd.AddCommand(statusCmd)
	authCmd.AddCommand(clearCmd)

	RootCmd.AddCommand(authCmd)
}