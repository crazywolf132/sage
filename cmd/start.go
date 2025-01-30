package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/crazywolf132/sage/internal/gitutils"
)

// startCmd represents "sage start <branch-name>"
var startCmd = &cobra.Command{
	Use:   "start <branch-name>",
	Short: "Create and switch to a new branch from the default branch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		branchName := args[0]

		// 1. Determine the default branch from config, fallback to "main"
		defaultBranch := viper.GetString("defaultBranch")
		if defaultBranch == "" {
			defaultBranch = "main"
		}

		// 2. Ensure working directory is clean or prompt user
		clean, err := gitutils.IsWorkingDirectoryClean()
		if err != nil {
			return err
		}
		if !clean {
			fmt.Println("\033[33mWARNING: You have uncommitted changes.\033[0m")
			// Optionally ask to stash or commit before proceeding.
		}

		// 3. Checkout default branch, pull latest
		if err := gitutils.RunGitCommand("switch", defaultBranch); err != nil {
			return err
		}
		if err := gitutils.RunGitCommand("pull"); err != nil {
			return err
		}

		// 4. Create new branch and switch
		if err := gitutils.RunGitCommand("switch", "-c", branchName); err != nil {
			return err
		}
		fmt.Printf("Switched to a new branch '%s'\n", branchName)

		// 5. If --push is set, push branch to origin
		doPush, _ := cmd.Flags().GetBool("push")
		if doPush {
			if err := gitutils.RunGitCommand("push", "-u", "origin", branchName); err != nil {
				return err
			}
			fmt.Printf("Branch '%s' pushed to origin\n", branchName)
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(startCmd)
	startCmd.Flags().Bool("push", false, "Immediately push the new branch to the remote")
}
