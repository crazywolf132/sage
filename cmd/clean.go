package cmd

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/sage/internal/gitutils"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
    Use:   "clean",
    Short: "Clean up merged branches",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Get default branch
        defaultBranch, err := gitutils.GetDefaultBranch()
        if err != nil {
            return fmt.Errorf("failed to determine default branch: %w", err)
        }

        // Get current branch
        currentBranch, err := gitutils.GetCurrentBranch()
        if err != nil {
            return fmt.Errorf("failed to get current branch: %w", err)
        }

        // Get merged branches
        mergedBranches, err := gitutils.GetMergedBranches(defaultBranch)
        if err != nil {
            return fmt.Errorf("failed to get merged branches: %w", err)
        }

        // Filter branches to delete
        var toDelete []string
        for _, branch := range mergedBranches {
            branch = strings.TrimSpace(branch)
            if branch == "" || branch == defaultBranch || branch == currentBranch {
                continue
            }
            toDelete = append(toDelete, branch)
        }

        if len(toDelete) == 0 {
            fmt.Println("✨ No merged branches to clean up")
            return nil
        }

        // Show branches to delete
        fmt.Println(ui.ColoredText("Branches to clean:", ui.Sage))
        for _, branch := range toDelete {
            fmt.Printf("  %s\n", branch)
        }

        // Confirm deletion
        confirm := false
        prompt := &survey.Confirm{
            Message: "Delete these branches?",
            Default: false,
        }
        if err := survey.AskOne(prompt, &confirm); err != nil {
            return err
        }

        if !confirm {
            fmt.Println("Cleanup cancelled")
            return nil
        }

        // Delete branches
        for _, branch := range toDelete {
            if err := gitutils.RunGitCommand("branch", "-d", branch); err != nil {
                fmt.Printf("⚠️ Failed to delete %s: %v\n", branch, err)
            } else {
                fmt.Printf("🗑️ Deleted %s\n", branch)
            }
        }

        return nil
    },
}

func init() {
    RootCmd.AddCommand(cleanCmd)
}
