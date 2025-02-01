package cmd

import (
	"github.com/spf13/cobra"
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Manage pull requests",
}

func init() {
	rootCmd.AddCommand(prCmd)
}
