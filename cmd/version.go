package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the version string for sage. For official releases, it is set via ldflags.
// For development builds, the version is set at build time via ldflags.
// The format is {major}.{minor}.{patch}-dev-{hour}.{minute}.{day}.{month}.{year}
// Example: 0.0.0-dev-15.04.02.01.2006
var Version = "0.0.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of sage",
	Run: func(cmd *cobra.Command, args []string) {
		short, _ := cmd.Flags().GetBool("short")
		if short {
			fmt.Println(Version)
		} else {
			fmt.Println("sage version:", Version)
		}
	},
}

func init() {
	// Add a flag for short version output
	versionCmd.Flags().BoolP("short", "s", false, "Display only the version number")

	// Register the version command under the root command
	rootCmd.AddCommand(versionCmd)

	// Set the version for the root command so that -v/--version flags work automatically
	rootCmd.Version = Version
}
