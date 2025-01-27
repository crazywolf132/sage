package cmd

import (
	"github.com/spf13/cobra"
)

// prCmd is the parent for all "sage pr" subcommands
var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Manage pull requests (create, list, checkout, merge, close, etc.)",
}

func init() {
	RootCmd.AddCommand(prCmd)
	// We'll add the subcommands in their respective .go files
}
