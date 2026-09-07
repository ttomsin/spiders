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
	Use:   "set-token [tokens...]",
	Short: "Store your __Secure-next-auth.session-token (.0, .1) securely on your machine",
	Long: `Save your ChatGPT authentication session cookies.
You can pass single tokens or multiple chunked cookies:
  chatgpt-spider auth set-token "__Secure-next-auth.session-token.0=<val0>" "__Secure-next-auth.session-token.1=<val1>"
Or pass them interactively.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cookieMap := make(map[string]string)

		if len(args) > 0 {
			for i, a := range args {
				trimmed := strings.TrimSpace(a)
				if strings.Contains(trimmed, "=") {
					kv := strings.SplitN(trimmed, "=", 2)
					cookieMap[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
				} else {
					// Positional: if 2 args without =, treat as .0 and .1
					if len(args) > 1 {
						cookieMap[fmt.Sprintf("__Secure-next-auth.session-token.%d", i)] = trimmed
					} else {
						cookieMap["__Secure-next-auth.session-token"] = trimmed
					}
				}
			}
		} else {
			fmt.Println("Enter ChatGPT session tokens. Press Enter on an empty prompt when done:")
			for i := 0; ; i++ {
				fmt.Printf("Token / Chunk .%d (input hidden): ", i)
				pass, err := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Println()
				if err != nil {
					return err
				}
				trimmed := strings.TrimSpace(string(pass))
				if trimmed == "" {
					break
				}
				cookieMap[fmt.Sprintf("__Secure-next-auth.session-token.%d", i)] = trimmed
			}
		}

		if len(cookieMap) == 0 {
			return fmt.Errorf("no tokens provided")
		}

		if err := auth.SaveSessionCookies(cookieMap); err != nil {
			return fmt.Errorf("failed to save cookies: %w", err)
		}

		fmt.Println(color.GreenString("✓ Successfully stored %d ChatGPT session cookie(s)!", len(cookieMap)))
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
		cookies, err := auth.GetSessionCookies()
		if err != nil {
			return err
		}

		fmt.Printf("%s: %s\n", cyan("Profile Storage"), profileDir)
		if len(cookies) == 0 {
			fmt.Printf("%s: %s\n", cyan("Session Cookies"), yellow("None saved (using browser profile cookies)"))
		} else {
			fmt.Printf("%s: %s (%d stored)\n", cyan("Session Cookies"), green("Stored"), len(cookies))
			for k, v := range cookies {
				fmt.Printf("  • %s: %s\n", k, auth.MaskToken(v))
			}
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