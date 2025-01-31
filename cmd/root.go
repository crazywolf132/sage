package cmd

import (
	"github.com/crazywolf132/sage/internal/config"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/crazywolf132/sage/internal/update"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sage",
	Short: "A slim, powerful Git helper CLI",
	Long: `Sage 2.0 streamlines Git workflows with minimal overhead.
Use subcommands like commit, clean, pr, etc.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Load config (global + local) once
		if err := config.LoadAllConfigs(); err != nil {
			ui.Warnf("Failed to load config: %v\n", err)
		}

		// (Optional) auto-update check, ignore errors
		_ = update.CheckForUpdates()
		return nil
	},
}

func init() {
	rootCmd.SetUsageTemplate(ui.ColorHeadings(rootCmd.UsageTemplate()))
}

// Execute is the root entrpoint
func Execute() error {
	return rootCmd.Execute()
}
