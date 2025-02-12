package cmd

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var stashCmd = &cobra.Command{
	Use:   "stash",
	Short: "Manage stashed changes in virtual branches",
	Long: `Manage stashed changes in virtual branches.

When switching between virtual branches, any uncommitted changes are automatically
stashed and associated with the branch. This command provides tools to manage
these stashed changes.`,
}

var stashListCmd = &cobra.Command{
	Use:   "list",
	Short: "List virtual branches with stashed changes",
	RunE: func(cmd *cobra.Command, args []string) error {
		branches, err := vbranchManager.ListVirtualBranches()
		if err != nil {
			return fmt.Errorf("failed to list virtual branches: %w", err)
		}

		hasStashed := false
		for _, vb := range branches {
			hasStash, err := vbranchManager.HasStashedChanges(vb.Name)
			if err != nil {
				return err
			}
			if hasStash {
				hasStashed = true
				status := " "
				if vb.Active {
					status = "*"
				}
				ui.Infof("%s %s (stashed changes)", status, vb.Name)
			}
		}

		if !hasStashed {
			ui.Info("No stashed changes found in any virtual branch")
		}
		return nil
	},
}

var stashPopCmd = &cobra.Command{
	Use:   "pop [branch-name]",
	Short: "Pop stashed changes from a virtual branch",
	RunE: func(cmd *cobra.Command, args []string) error {
		var branchName string
		if len(args) > 0 {
			branchName = args[0]
		} else {
			// List branches with stashed changes
			branches, err := vbranchManager.ListVirtualBranches()
			if err != nil {
				return fmt.Errorf("failed to list virtual branches: %w", err)
			}

			var options []string
			for _, vb := range branches {
				hasStash, err := vbranchManager.HasStashedChanges(vb.Name)
				if err != nil {
					return err
				}
				if hasStash {
					status := " "
					if vb.Active {
						status = "*"
					}
					options = append(options, fmt.Sprintf("%s %s", status, vb.Name))
				}
			}

			if len(options) == 0 {
				ui.Info("No stashed changes found in any virtual branch")
				return nil
			}

			var selected string
			prompt := &survey.Select{
				Message: "Select a branch to pop stashed changes from:",
				Options: options,
			}
			survey.AskOne(prompt, &selected)
			branchName = strings.Split(strings.TrimSpace(selected), " ")[1]
		}

		// Check if branch has stashed changes
		hasStash, err := vbranchManager.HasStashedChanges(branchName)
		if err != nil {
			return err
		}
		if !hasStash {
			return fmt.Errorf("no stashed changes found in branch %s", branchName)
		}

		// Pop the stashed changes
		if err := vbranchManager.PopStashedChanges(branchName); err != nil {
			return fmt.Errorf("failed to pop stashed changes: %w", err)
		}

		ui.Successf("Successfully popped stashed changes from '%s'", branchName)
		return nil
	},
}

var stashDropCmd = &cobra.Command{
	Use:   "drop [branch-name]",
	Short: "Drop stashed changes from a virtual branch",
	RunE: func(cmd *cobra.Command, args []string) error {
		var branchName string
		if len(args) > 0 {
			branchName = args[0]
		} else {
			// List branches with stashed changes
			branches, err := vbranchManager.ListVirtualBranches()
			if err != nil {
				return fmt.Errorf("failed to list virtual branches: %w", err)
			}

			var options []string
			for _, vb := range branches {
				hasStash, err := vbranchManager.HasStashedChanges(vb.Name)
				if err != nil {
					return err
				}
				if hasStash {
					status := " "
					if vb.Active {
						status = "*"
					}
					options = append(options, fmt.Sprintf("%s %s", status, vb.Name))
				}
			}

			if len(options) == 0 {
				ui.Info("No stashed changes found in any virtual branch")
				return nil
			}

			var selected string
			prompt := &survey.Select{
				Message: "Select a branch to drop stashed changes from:",
				Options: options,
			}
			survey.AskOne(prompt, &selected)
			branchName = strings.Split(strings.TrimSpace(selected), " ")[1]
		}

		// Confirm before dropping
		if !ui.Confirm(fmt.Sprintf("Are you sure you want to drop stashed changes from '%s'?", branchName)) {
			ui.Info("Operation cancelled")
			return nil
		}

		// Drop the stashed changes
		if err := vbranchManager.DropStashedChanges(branchName); err != nil {
			return fmt.Errorf("failed to drop stashed changes: %w", err)
		}

		ui.Successf("Successfully dropped stashed changes from '%s'", branchName)
		return nil
	},
}

func init() {
	vbranchCmd.AddCommand(stashCmd)
	stashCmd.AddCommand(stashListCmd)
	stashCmd.AddCommand(stashPopCmd)
	stashCmd.AddCommand(stashDropCmd)
}
