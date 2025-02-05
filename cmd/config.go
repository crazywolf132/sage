package cmd

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/config"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Sage configuration",
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Args:  cobra.ExactArgs(1),
	Short: "Get a config value",
	RunE: func(cmd *cobra.Command, args []string) error {
		val := config.Get(args[0])
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
	Short: "Set a config value (by default in local config if in a repo, otherwise global)",
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]
		err := config.Set(key, value)
		if err != nil {
			return err
		}
		fmt.Printf("%s %s=%s\n", ui.Green("Set"), key, value)
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available configuration properties",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("\n%s\n", ui.Sage("Available Configuration Properties:"))

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
		fmt.Printf("  List values:   %s\n\n", ui.White("sage config list"))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
}
