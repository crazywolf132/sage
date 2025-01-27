package cmd

import (
	"fmt"
	"strings"

	"github.com/crazywolf132/sage/internal/gitutils"
	"github.com/spf13/cobra"
)

// pushCmd represents "sage push"
var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push the current branch to the remote",
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		currentBranch, err := gitutils.GetCurrentBranch()
		if err != nil {
			return err
		}

		if force {
			// Confirm destructive action
			confirm, _ := cmd.Flags().GetBool("yes")
			if !confirm {
				// This is a quick text-based "are you sure?" approach.
				// In production, you might want a more robust prompt or skip if --yes is provided.
				fmt.Printf("You are about to force push branch '%s'. Type 'yes' to confirm: ", currentBranch)
				fmt.Print("Do you want to continue? [y/N]: ")
				var userInput string
				if _, err := fmt.Scanln(&userInput); err != nil {
					return fmt.Errorf("failed to read user input: %w", err)
				}
				if strings.ToLower(userInput) != "yes" {
					fmt.Println("Force push aborted.")
					return nil
				}
			}

			// Create a backup reference before force push
			backupRef := fmt.Sprintf("sage/backup/%s", currentBranch)
			if err := gitutils.RunGitCommand("tag", backupRef); err == nil {
				fmt.Printf("Created backup reference: %s\n", backupRef)
			}

			if err := gitutils.RunGitCommand("push", "--force", "origin", currentBranch); err != nil {
				return err
			}
			fmt.Printf("Force push completed for branch '%s'\n", currentBranch)
			return nil
		}

		// Normal push
		if err := gitutils.RunGitCommand("push", "origin", currentBranch); err != nil {
			return err
		}
		fmt.Printf("Pushed branch '%s' to origin\n", currentBranch)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(pushCmd)
	pushCmd.Flags().BoolP("force", "f", false, "Force push the current branch")
	pushCmd.Flags().Bool("yes", false, "Skip confirmation for force push (use with caution)")
}
