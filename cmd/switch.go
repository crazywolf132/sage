package cmd

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/sage/internal/gitutils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// switchCmd represents "sage switch [branch-name]"
var switchCmd = &cobra.Command{
	Use:   "switch [branch-name]",
	Short: "Switch to an existing branch or create a new one",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no branch name provided, show interactive branch selection
		if len(args) == 0 {
			// Get list of all branches
			branches, err := gitutils.GetBranches()
			if err != nil {
				return err
			}

			// Prepare the survey
			var selectedBranch string
			prompt := &survey.Select{
				Message: "Choose a branch to switch to:",
				Options: branches,
			}

			// Show the interactive prompt
			if err := survey.AskOne(prompt, &selectedBranch); err != nil {
				return err
			}

			// Switch to the selected branch
			if err := gitutils.RunGitCommand("checkout", selectedBranch); err != nil {
				return err
			}

			fmt.Printf("Switched to branch '%s'\n", selectedBranch)
			return nil
		}

		// Branch name provided
		branchName := args[0]

		// Check if branch exists
		exists, err := gitutils.BranchExists(branchName)
		if err != nil {
			return err
		}

		if exists {
			// Switch to existing branch
			if err := gitutils.RunGitCommand("checkout", branchName); err != nil {
				return err
			}
			fmt.Printf("Switched to branch '%s'\n", branchName)
			return nil
		}

		// Branch doesn't exist, ask if user wants to create it
		var createBranch bool
		prompt := &survey.Confirm{
			Message: fmt.Sprintf("Branch '%s' doesn't exist. Create it?", branchName),
		}

		if err := survey.AskOne(prompt, &createBranch); err != nil {
			return err
		}

		if !createBranch {
			fmt.Println("Operation cancelled.")
			return nil
		}

		// Create new branch from default branch
		defaultBranch := viper.GetString("defaultBranch")
		if defaultBranch == "" {
			defaultBranch = "main"
		}

		// Checkout default branch and pull latest
		if err := gitutils.RunGitCommand("checkout", defaultBranch); err != nil {
			return err
		}
		if err := gitutils.RunGitCommand("pull"); err != nil {
			return err
		}

		// Create and switch to new branch
		if err := gitutils.RunGitCommand("checkout", "-b", branchName); err != nil {
			return err
		}
		fmt.Printf("Created and switched to new branch '%s'\n", branchName)

		return nil
	},
}

func init() {
	RootCmd.AddCommand(switchCmd)
}
