package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "0.1.0" // Replace or override at build time if desired

// versionCmd represents "sage version"
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of Sage",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Sage version %s\n", version)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
