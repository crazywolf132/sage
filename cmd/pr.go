package cmd

import (
	"github.com/spf13/cobra"
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Manage pull requests",
	RunE: func(cmd *cobra.Command, args []string) error {
		// If --help is provided, show help
		if cmd.Flags().Changed("help") {
			return cmd.Help()
		}
		// Otherwise, run pr status
		return prStatusCmd.RunE(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(prCmd)
}
