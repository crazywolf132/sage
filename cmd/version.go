package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "2.0.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show Sage version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Sage version %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
