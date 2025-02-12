package cmd

import (
	"fmt"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

// Version string is set during build via ldflags
var Version string

var (
	once          sync.Once
	versionString string
)

// GetVersion returns the version string, determining it on first call
func GetVersion() string {
	once.Do(func() {
		versionString = determineVersion()
	})
	return versionString
}

// determineVersion returns the version string, falling back to build info for development builds
func determineVersion() string {
	// If version was set by ldflags (during release builds), use it
	if Version != "" {
		return Version
	}

	// Try to get version from build info (works for go install)
	if info, ok := debug.ReadBuildInfo(); ok {
		// First check the main module version
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return strings.TrimPrefix(info.Main.Version, "v")
		}

		// Check for vcs information
		var revision, time string
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				revision = setting.Value[:7] // Short SHA
			case "vcs.time":
				time = setting.Value
			}
		}
		if revision != "" && time != "" {
			return fmt.Sprintf("dev-%s-%s", revision, time)
		}
	}

	return "0.0.0-dev"
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of sage",
	Run: func(cmd *cobra.Command, args []string) {
		short, _ := cmd.Flags().GetBool("short")
		if short {
			fmt.Println(GetVersion())
		} else {
			fmt.Println("sage version:", GetVersion())
		}
	},
}

func init() {
	// Add a flag for short version output
	versionCmd.Flags().BoolP("short", "s", false, "Display only the version number")

	// Register the version command under the root command
	rootCmd.AddCommand(versionCmd)

	// Set the version for the root command so that -v/--version flags work automatically
	rootCmd.Version = GetVersion()
}
