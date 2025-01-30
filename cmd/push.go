package cmd

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/sage/internal/gitutils"
	"github.com/spf13/cobra"
)

// pushCmd represents "sage push"
var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push changes to remote",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get current branch
		currentBranch, err := gitutils.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}

		// Check if force push is requested
		force, _ := cmd.Flags().GetBool("force")
		skipConfirm, _ := cmd.Flags().GetBool("yes")

		if force {
			if !skipConfirm {
				fmt.Printf("⚠️  Warning: You're about to force push to '%s'\n", currentBranch)
				fmt.Println("   This will overwrite remote history!")

				var confirm bool
				prompt := &survey.Confirm{
					Message: "Are you sure you want to continue?",
				}
				if err := survey.AskOne(prompt, &confirm); err != nil {
					return err
				}
				if !confirm {
					fmt.Println("   Operation cancelled")
					return nil
				}
			}

			fmt.Printf("\n🔄 Force pushing to '%s'...\n", currentBranch)

			// Create a backup reference before force push
			backupRef := fmt.Sprintf("sage/backup/%s", currentBranch)
			if err := gitutils.RunGitCommand("tag", backupRef); err == nil {
				fmt.Printf("   💾 Created backup reference: %s\n", backupRef)
			}

			if err := gitutils.RunGitCommand("push", "--force", "origin", currentBranch); err != nil {
				return fmt.Errorf("failed to force push: %w", err)
			}
		} else {
			fmt.Printf("\n🔄 Pushing to '%s'...\n", currentBranch)
			if err := gitutils.RunGitCommand("push", "origin", currentBranch); err != nil {
				return fmt.Errorf("failed to push: %w", err)
			}
		}

		fmt.Printf("\n✨ Changes published!\n")
		fmt.Printf("   Branch '%s' is up to date\n", currentBranch)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(pushCmd)
	pushCmd.Flags().BoolP("force", "f", false, "Force push the current branch")
	pushCmd.Flags().Bool("yes", false, "Skip confirmation for force push (use with caution)")
}
