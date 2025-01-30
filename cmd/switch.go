package cmd

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/sage/internal/gitutils"
	"github.com/spf13/cobra"
)

// switchCmd represents "sage switch [branch-name]"
var switchCmd = &cobra.Command{
	Use:   "switch [branch-name]",
	Short: "Switch to an existing branch or create a new one",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no branch name provided, show interactive branch selection
		if len(args) == 0 {
			branches, err := gitutils.GetBranches()
			if err != nil {
				return err
			}

			var selectedBranch string
			prompt := &survey.Select{
				Message: "Choose a branch to switch to:",
				Options: branches,
			}

			if err := survey.AskOne(prompt, &selectedBranch); err != nil {
				return err
			}

			args = []string{selectedBranch}
		}

		// Branch name provided
		branchName := args[0]

		// Check if branch exists
		exists, err := gitutils.BranchExists(branchName)
		if err != nil {
			return err
		}

		if exists {
			fmt.Printf("\n🔄 Switching to '%s'...\n", branchName)
			// Switch to existing branch
			if err := gitutils.RunGitCommand("switch", branchName); err != nil {
				return fmt.Errorf("failed to switch branch: %w", err)
			}

			// Check if upstream exists and pull
			if upstream, err := gitutils.DefaultRunner.RunGitCommandWithOutput("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"); err == nil {
				fmt.Println("   ⬇️  Pulling latest changes")
				if err := gitutils.RunGitCommand("pull"); err != nil {
					fmt.Printf("   ⚠️  Failed to pull from %s\n", strings.TrimSpace(upstream))
				}
			}

			fmt.Printf("\n✨ Switched to branch!\n")
			fmt.Printf("   %s\n", branchName)
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
			fmt.Println("   Operation cancelled")
			return nil
		}

		// Create new branch from default branch
		defaultBranch, err := gitutils.GetDefaultBranch()
		if err != nil {
			fmt.Println("⚠️  Could not determine default branch, using 'main'")
			defaultBranch = "main"
		}

		fmt.Printf("\n🔄 Creating new branch...\n")

		// Checkout default branch and pull latest
		fmt.Printf("   ⎇  Switching to %s\n", defaultBranch)
		if err := gitutils.RunGitCommand("switch", defaultBranch); err != nil {
			return fmt.Errorf("failed to switch to default branch: %w", err)
		}

		fmt.Println("   ⬇️  Pulling latest changes")
		if err := gitutils.RunGitCommand("pull"); err != nil {
			return fmt.Errorf("failed to pull latest changes: %w", err)
		}

		// Create and switch to new branch
		fmt.Printf("   🌱 Creating branch '%s'\n", branchName)
		if err := gitutils.RunGitCommand("switch", "-c", branchName); err != nil {
			return fmt.Errorf("failed to create branch: %w", err)
		}

		fmt.Printf("\n✨ Branch created and switched!\n")
		fmt.Printf("   %s\n", branchName)

		return nil
	},
}

func init() {
	RootCmd.AddCommand(switchCmd)
}
