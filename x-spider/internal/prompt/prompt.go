package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"golang.org/x/term"
	"x-spider/internal/auth"
	"x-spider/internal/config"
)

// AskQuestionsPrompts asks for required missing configuration values interactively
func AskQuestionsPrompts(cfg *config.Config) error {
	reader := bufio.NewReader(os.Stdin)

	// 1. Auth Token
	if cfg.AuthToken == "" {
		for {
			fmt.Print("What's your Twitter auth token? ")
			bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Println() // newline after password input
			if err != nil {
				return fmt.Errorf("failed to read auth token: %w", err)
			}
			token := strings.TrimSpace(string(bytePassword))
			if len(token) >= 20 {
				cfg.AuthToken = token

				// Offer to save credentials securely
				fmt.Print("Save this token securely on your computer for future sessions? [Y/n]: ")
				ans, _ := reader.ReadString('\n')
				ans = strings.TrimSpace(strings.ToLower(ans))
				if ans == "" || ans == "y" || ans == "yes" {
					if err := auth.SaveAuthToken(token); err == nil {
						green := color.New(color.FgGreen).SprintfFunc()
						fmt.Printf("%s\n\n", green("✓ Auth token securely saved to %s", auth.GetStorageFilePath()))
					}
				}
				break
			}
			fmt.Println("Please enter a valid Twitter auth token (min 20 characters).")
		}
	}

	// 2. Search Keyword or Thread URL
	if cfg.SearchKeyword == "" && cfg.ThreadURL == "" {
		for {
			fmt.Print("What's the search keyword? ")
			line, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read keyword: %w", err)
			}
			keyword := strings.TrimSpace(line)
			if keyword != "" {
				cfg.SearchKeyword = keyword
				break
			}
			fmt.Println("Please enter a non-empty search keyword.")
		}
	}

	// 3. Limit
	if cfg.Limit <= 0 {
		for {
			fmt.Print("How many tweets do you want to crawl? [default: 10]: ")
			line, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read limit: %w", err)
			}
			input := strings.TrimSpace(line)
			if input == "" {
				cfg.Limit = 10
				break
			}
			val, err := strconv.Atoi(input)
			if err == nil && val > 0 {
				cfg.Limit = val
				break
			}
			fmt.Println("Please enter a number greater than 0.")
		}
	}

	// 4. Export format
	if cfg.ExportFormat == "" {
		fmt.Print("What format do you want to export? (1: CSV, 2: Excel XLSX, 3: JSON) [default: 1]: ")
		line, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(line)
		if choice == "2" || strings.ToLower(choice) == "xlsx" {
			cfg.ExportFormat = "xlsx"
		} else if choice == "3" || strings.ToLower(choice) == "json" {
			cfg.ExportFormat = "json"
		} else {
			cfg.ExportFormat = "csv"
		}
	}

	return nil
}
