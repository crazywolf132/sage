package cmd

import (
	"github.com/crazywolf132/sage/internal/config"
	"github.com/crazywolf132/sage/internal/gh"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/crazywolf132/sage/internal/update"
	"github.com/crazywolf132/sage/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sage",
	Short: "Burning away Git complexity",
	Long: `Sage is a modern CLI tool that simplifies Git workflows and enhances productivity.

It provides intuitive commands for common Git operations and adds powerful features like:
• Smart commit messages with AI assistance
• Streamlined PR workflows and status checks
• Easy branch synchronization
• GitHub integration
• Interactive UI elements for better user experience

Run 'sage help' to see all available commands or 'sage <command> --help' for detailed information about a specific command.`,
	Version:       version.Get(),
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Load config (global + local) once
		if err := config.LoadAllConfigs(); err != nil {
			ui.Warnf("Failed to load config: %v\n", err)
		}

		// Check for updates before running any command
		ghClient := gh.NewClient()
		_ = update.CheckForUpdates(ghClient, version.Get())
	},
}

func init() {
	rootCmd.SetUsageTemplate(ui.ColorHeadings(rootCmd.UsageTemplate()))
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}
