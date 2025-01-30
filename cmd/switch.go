package cmd

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/spf13/cobra"
)

// switchCmd represents "sage switch [branch-name]"
var switchCmd = &cobra.Command{
	Use:   "switch [branch-name]",
	Short: "Switch to an existing branch or create a new one",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, err := getRepo()
		if err != nil {
			return err
		}

		// If no branch name provided, show interactive branch selection
		if len(args) == 0 {
			branches, err := repo.ListBranches()
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
		exists, err := repo.BranchExists(branchName, false)
		if err != nil {
			return err
		}

		if exists {
			fmt.Printf("\n🔄 Switching to '%s'...\n", branchName)
			// Switch to existing branch
			if _, err := repo.Switch(&git.SwitchOpts{
				Name: branchName,
			}); err != nil {
				return fmt.Errorf("failed to switch branch: %w", err)
			}

			// Check if upstream exists and pull
			if upstream, err := repo.HasRemote(branchName); err == nil && upstream {
				fmt.Println("   ⬇️  Pulling latest changes")
				if err := repo.Pull(); err != nil {
					fmt.Println("   ⚠️  Failed to pull from remote")
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
		defaultBranch, err := repo.DefaultBranch()
		if err != nil {
			fmt.Println("⚠️  Could not determine default branch, using 'main'")
			defaultBranch = "main"
		}

		fmt.Printf("\n🔄 Creating new branch...\n")

		// Checkout default branch and pull latest
		fmt.Printf("   ⎇  Switching to %s\n", defaultBranch)
		if _, err := repo.Switch(&git.SwitchOpts{Name: defaultBranch}); err != nil {
			return fmt.Errorf("failed to switch to default branch: %w", err)
		}

		fmt.Println("   ⬇️  Pulling latest changes")
		if err := repo.Pull(); err != nil {
			return fmt.Errorf("failed to pull latest changes: %w", err)
		}

		// Create and switch to new branch
		fmt.Printf("   🌱 Creating branch '%s'\n", branchName)
		if _, err := repo.Switch(&git.SwitchOpts{
			Create: true,
			Name:   branchName,
		}); err != nil {
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
