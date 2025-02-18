package cmd

import (
	"fmt"
	"strings"

	"github.com/crazywolf132/sage/internal/config"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var (
	useLocalConfig bool
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Sage configuration",
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Args:  cobra.ExactArgs(1),
	Short: "Get a config value",
	Long: `Get a configuration value. By default, reads from the global config.
Use --local to read from the local repository config instead (only works inside git repositories).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if useLocalConfig {
			g := git.NewShellGit()
			inRepo, _ := g.IsRepo()
			if !inRepo {
				return fmt.Errorf("--local flag can only be used inside a git repository")
			}
		}

		val := config.Get(args[0], useLocalConfig)
		if val == "" {
			fmt.Println(ui.Gray("not set"))
		} else {
			fmt.Println(val)
		}
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Args:  cobra.ExactArgs(2),
	Short: "Set a config value",
	Long: `Set a configuration value. By default, saves to the global config.
Use --local to save to the local repository config instead (only works inside git repositories).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if useLocalConfig {
			g := git.NewShellGit()
			inRepo, _ := g.IsRepo()
			if !inRepo {
				return fmt.Errorf("--local flag can only be used inside a git repository")
			}
		}

		key, value := args[0], args[1]
		err := config.Set(key, value, !useLocalConfig)
		if err != nil {
			return err
		}
		location := "global"
		if useLocalConfig {
			location = "local"
		}
		fmt.Printf("%s %s=%s (%s config)\n", ui.Green("Set"), key, value, location)
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available configuration properties",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("\n%s\n", ui.Sage("Available Configuration Properties:"))
		fmt.Printf("%s\n", ui.Gray("Use 'sage config experimental' to view experimental features"))

		// AI Configuration
		fmt.Printf("\n%s\n", ui.Bold("AI Settings:"))
		fmt.Printf("  %s\n    %s\n    %s %s\n",
			ui.White("ai.model"),
			"The AI model to use for generating content",
			"Default:", ui.Gray("gpt-4"))
		fmt.Printf("  %s\n    %s\n    %s %s\n",
			ui.White("ai.base_url"),
			"Base URL for the AI API endpoint",
			"Default:", ui.Gray("https://api.openai.com/v1"))
		fmt.Printf("  %s\n    %s\n    %s %s\n",
			ui.White("ai.api_key"),
			"API key for the AI service (can also be set via OPENAI_API_KEY env var)",
			"Default:", ui.Gray("none"))

		// Git Configuration
		fmt.Printf("\n%s\n", ui.Bold("Git Settings:"))
		fmt.Printf("  %s\n    %s\n    %s %s\n",
			ui.White("git.default_branch"),
			"Default branch to use when creating PRs or syncing",
			"Default:", ui.Gray("main"))
		fmt.Printf("  %s\n    %s\n    %s %s\n",
			ui.White("git.merge_method"),
			"Default merge method for PRs (merge, squash, rebase)",
			"Default:", ui.Gray("merge"))

		// GitHub Configuration
		fmt.Printf("\n%s\n", ui.Bold("GitHub Settings:"))
		fmt.Printf("  %s\n    %s\n    %s %s\n",
			ui.White("github.token"),
			"GitHub personal access token (can also be set via SAGE_GITHUB_TOKEN or GITHUB_TOKEN env vars)",
			"Default:", ui.Gray("none"))

		// PR Configuration
		fmt.Printf("\n%s\n", ui.Bold("Pull Request Settings:"))
		fmt.Printf("  %s\n    %s\n    %s %s\n",
			ui.White("pr.draft"),
			"Whether to create PRs as drafts by default",
			"Default:", ui.Gray("false"))
		fmt.Printf("  %s\n    %s\n    %s %s\n",
			ui.White("pr.reviewers"),
			"Default reviewers to assign to PRs (comma-separated)",
			"Default:", ui.Gray("none"))
		fmt.Printf("  %s\n    %s\n    %s %s\n",
			ui.White("pr.labels"),
			"Default labels to apply to PRs (comma-separated)",
			"Default:", ui.Gray("none"))

		fmt.Printf("\n%s\n", ui.Bold("Usage:"))
		fmt.Printf("  Set a value:   %s\n", ui.White("sage config set <key> <value>"))
		fmt.Printf("  Get a value:   %s\n", ui.White("sage config get <key>"))
		fmt.Printf("  Remove a value: %s\n", ui.White("sage config unset <key>"))
		fmt.Printf("  List values:   %s\n", ui.White("sage config list"))
		fmt.Printf("  View experimental: %s\n\n", ui.White("sage config experimental"))

		return nil
	},
}

var configUnsetCmd = &cobra.Command{
	Use:   "unset <key>",
	Args:  cobra.ExactArgs(1),
	Short: "Remove a config value",
	Long: `Remove a configuration value. By default, removes from the global config.
Use --local to remove from the local repository config instead (only works inside git repositories).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if useLocalConfig {
			g := git.NewShellGit()
			inRepo, _ := g.IsRepo()
			if !inRepo {
				return fmt.Errorf("--local flag can only be used inside a git repository")
			}
		}

		key := args[0]
		err := config.Unset(key, !useLocalConfig)
		if err != nil {
			return err
		}
		location := "global"
		if useLocalConfig {
			location = "local"
		}
		fmt.Printf("%s Removed %s (%s config)\n", ui.Green("✓"), key, location)
		return nil
	},
}

var configExperimentalCmd = &cobra.Command{
	Use:   "experimental",
	Short: "Show experimental features and their status",
	Long: `Display all available experimental features and their current status.
Shows whether each feature is enabled globally or locally, and provides
information about what each feature does.`,
	Aliases: []string{"exp", "x"},
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("\n%s\n", ui.Sage("✨ Experimental Features"))
		fmt.Printf("%s\n", ui.Gray("Features that are being tested and may change or be removed."))
		fmt.Println()

		// Check if we're in a git repo
		g := git.NewShellGit()
		inRepo, _ := g.IsRepo()

		// Get all experimental features
		features := config.GetExperimentalFeatures()

		// Calculate the longest feature name for alignment
		maxLen := 0
		for name := range features {
			if len(name) > maxLen {
				maxLen = len(name)
			}
		}

		// Print each feature
		for name, feature := range features {
			// Get enabled status (global and local)
			globalEnabled := config.Get("experimental."+name, false) == "true"
			localEnabled := inRepo && config.Get("experimental."+name, true) == "true"

			// Feature name
			fmt.Printf("  %s%s", ui.White(name), strings.Repeat(" ", maxLen-len(name)))

			// Status indicators
			if feature.SageWide {
				if globalEnabled {
					fmt.Printf(" %s", ui.Green("● Enabled (All Sage repos)"))
				} else if localEnabled {
					fmt.Printf(" %s", ui.Blue("● Enabled (This repo)"))
				} else {
					fmt.Printf(" %s", ui.Gray("○ Disabled"))
				}
			} else {
				if localEnabled {
					fmt.Printf(" %s", ui.Blue("● Enabled"))
				} else {
					fmt.Printf(" %s", ui.Gray("○ Disabled"))
				}
			}

			// Feature description
			fmt.Printf("\n    %s\n", ui.Gray(getFeatureDescription(name)))

			// Git config info
			if feature.Key != "" {
				fmt.Printf("    %s %s\n", ui.White("Git Config:"), ui.Gray(feature.Key+"="+feature.Value))
			}

			// Usage examples
			if feature.SageWide {
				fmt.Printf("    %s\n      %s\n      %s\n",
					ui.White("Usage:"),
					ui.Gray("sage config set experimental."+name+" true      # Enable for all Sage usage"),
					ui.Gray("sage config set --local experimental."+name+" true  # Enable for this repo only"))
			} else {
				fmt.Printf("    %s\n      %s\n",
					ui.White("Usage:"),
					ui.Gray("sage config set --local experimental."+name+" true"))
			}
			fmt.Println()
		}

		return nil
	},
}

// getFeatureDescription returns a user-friendly description of an experimental feature
func getFeatureDescription(name string) string {
	switch name {
	case "rerere":
		return "Reuse Recorded Resolution - Git will remember how you resolved conflicts and automatically reuse those resolutions."
	case "commit-graph":
		return "Write commit graph on fetch - Significantly speeds up git log operations and commit traversal in large repositories."
	case "fsmonitor":
		return "File System Monitor - Speeds up git status operations by using OS-level file monitoring. Particularly effective in large repositories."
	case "maintenance":
		return "Git Auto-Maintenance - Automatically optimizes your repository's performance in the background:\n" +
			"    • Hourly prefetch: Keeps your repository up-to-date with remote changes\n" +
			"    • Loose object cleanup: Improves storage efficiency by packing loose objects\n" +
			"    • Daily reference packing: Organizes references (branches/tags) for faster access\n" +
			"    • Incremental repack: Maintains optimal repository storage\n" +
			"    \n" +
			"    This feature is especially useful for:\n" +
			"    • Large repositories with many objects\n" +
			"    • Teams with frequent commits and merges\n" +
			"    • Repositories with long history\n" +
			"    • Improving git command response times"
	default:
		return "No description available."
	}
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configUnsetCmd)
	configCmd.AddCommand(configExperimentalCmd)

	// Add --local flag to get, set, and unset commands
	configGetCmd.Flags().BoolVarP(&useLocalConfig, "local", "l", false, "Use local repository config")
	configSetCmd.Flags().BoolVarP(&useLocalConfig, "local", "l", false, "Use local repository config")
	configUnsetCmd.Flags().BoolVarP(&useLocalConfig, "local", "l", false, "Use local repository config")
}
