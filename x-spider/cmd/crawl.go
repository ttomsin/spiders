package cmd

import (
	"github.com/spf13/cobra"
)

// crawlCmd explicitly runs the crawler
var crawlCmd = &cobra.Command{
	Use:   "crawl",
	Short: "Crawl tweets matching keywords or thread URL",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RootCmd.RunE(cmd, args)
	},
}

func init() {
	RootCmd.AddCommand(crawlCmd)
}
