package cmd

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/crazywolf132/sage/internal/vbranch"
	"github.com/spf13/cobra"
)

var (
	vbranchManager vbranch.Manager
	vbranchWatcher *vbranch.Watcher
)

func initVirtualBranches() error {
	gitService := git.NewShellGit()

	var err error
	vbranchManager, err = vbranch.NewManager(gitService)
	if err != nil {
		return fmt.Errorf("failed to create virtual branch manager: %w", err)
	}

	vbranchWatcher, err = vbranch.NewWatcher(vbranchManager)
	if err != nil {
		return fmt.Errorf("failed to create virtual branch watcher: %w", err)
	}

	if err := vbranchWatcher.Start(); err != nil {
		return fmt.Errorf("failed to start virtual branch watcher: %w", err)
	}

	return nil
}

var vbranchCmd = &cobra.Command{
	Use:   "vbranch",
	Short: "Manage virtual branches",
	Long: `Virtual branches allow you to work on multiple features simultaneously
without switching Git branches. Changes are tracked separately and can be
materialized into real Git branches when ready.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := initVirtualBranches(); err != nil {
			return err
		}
		return nil
	},
}

var createVbranchCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new virtual branch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		baseBranch, _ := cmd.Flags().GetString("base")

		if baseBranch == "" {
			gitService := git.NewShellGit()
			var err error
			baseBranch, err = gitService.CurrentBranch()
			if err != nil {
				return fmt.Errorf("failed to get current branch: %w", err)
			}
		}

		vb, err := vbranchManager.CreateVirtualBranch(name, baseBranch)
		if err != nil {
			return fmt.Errorf("failed to create virtual branch: %w", err)
		}

		// Set this as the active branch in the watcher
		vbranchWatcher.SetActiveBranch(name)

		fmt.Printf("Created virtual branch '%s' based on '%s'\n", vb.Name, vb.BaseBranch)
		return nil
	},
}

var listVbranchCmd = &cobra.Command{
	Use:   "list",
	Short: "List all virtual branches",
	RunE: func(cmd *cobra.Command, args []string) error {
		branches, err := vbranchManager.ListVirtualBranches()
		if err != nil {
			return fmt.Errorf("failed to list virtual branches: %w", err)
		}

		if len(branches) == 0 {
			fmt.Println("No virtual branches found")
			return nil
		}

		for _, vb := range branches {
			status := " "
			if vb.Active {
				status = "*"
			}
			fmt.Printf("%s %-20s (based on %s, %d changes)\n", status, vb.Name, vb.BaseBranch, len(vb.Changes))
		}
		return nil
	},
}

var switchVbranchCmd = &cobra.Command{
	Use:     "switch [name]",
	Short:   "Switch to a virtual branch",
	Aliases: []string{"s", "sw"},
	RunE: func(cmd *cobra.Command, args []string) error {
		var name string
		if len(args) > 0 {
			name = args[0]
		} else {
			// Interactive branch selection
			branches, err := vbranchManager.ListVirtualBranches()
			if err != nil {
				return fmt.Errorf("failed to list virtual branches: %w", err)
			}

			if len(branches) == 0 {
				ui.Info("No virtual branches found")
				if ui.Confirm("Would you like to create one?") {
					return createVbranchCmd.RunE(cmd, []string{"feature-" + ui.AskString("Branch name: ")})
				}
				return nil
			}

			// Build branch options with status indicators
			options := make([]string, len(branches))
			for i, vb := range branches {
				status := ""
				if vb.Active {
					status = "* "
				}
				hasStash, _ := vbranchManager.HasStashedChanges(vb.Name)
				stashIndicator := ""
				if hasStash {
					stashIndicator = " [stashed]"
				}
				options[i] = fmt.Sprintf("%s%s (%d changes)%s", status, vb.Name, len(vb.Changes), stashIndicator)
			}

			var selected string
			prompt := &survey.Select{
				Message: "Select branch to switch to:",
				Options: options,
			}
			if err := survey.AskOne(prompt, &selected); err != nil {
				return err
			}

			// Extract branch name from selection
			name = strings.Split(strings.TrimLeft(selected, "* "), " ")[0]
		}

		// Check if branch exists
		if _, err := vbranchManager.GetVirtualBranch(name); err != nil {
			// Branch doesn't exist, offer to create it
			if ui.Confirm(fmt.Sprintf("Branch '%s' doesn't exist. Create it?", name)) {
				return createVbranchCmd.RunE(cmd, []string{name})
			}
			return nil
		}

		if err := vbranchManager.ApplyVirtualBranch(name); err != nil {
			return fmt.Errorf("failed to switch to virtual branch: %w", err)
		}

		vbranchWatcher.SetActiveBranch(name)
		ui.Successf("Switched to virtual branch '%s'", name)

		// Show branch status after switch
		return statusVbranchCmd.RunE(cmd, nil)
	},
}

var focusVbranchCmd = &cobra.Command{
	Use:     "focus",
	Short:   "Quickly switch between virtual branches with a fuzzy finder",
	Aliases: []string{"f"},
	RunE: func(cmd *cobra.Command, args []string) error {
		branches, err := vbranchManager.ListVirtualBranches()
		if err != nil {
			return fmt.Errorf("failed to list virtual branches: %w", err)
		}

		if len(branches) == 0 {
			ui.Info("No virtual branches found")
			if ui.Confirm("Would you like to create one?") {
				return newVbranchCmd.RunE(cmd, args)
			}
			return nil
		}

		// Build rich branch descriptions
		options := make([]string, len(branches))
		for i, vb := range branches {
			status := ""
			if vb.Active {
				status = ui.Green("* ")
			}
			hasStash, _ := vbranchManager.HasStashedChanges(vb.Name)
			stashIndicator := ""
			if hasStash {
				stashIndicator = ui.Yellow(" [stashed]")
			}
			changeCount := ""
			if len(vb.Changes) > 0 {
				changeCount = ui.Blue(fmt.Sprintf(" (%d changes)", len(vb.Changes)))
			}
			options[i] = fmt.Sprintf("%s%s%s%s\n  Based on: %s",
				status, vb.Name, changeCount, stashIndicator, vb.BaseBranch)
		}

		var selected string
		prompt := &survey.Select{
			Message: "Select branch to focus on:",
			Options: options,
		}
		if err := survey.AskOne(prompt, &selected); err != nil {
			return err
		}

		// Extract branch name from selection
		name := strings.Split(strings.TrimLeft(selected, "* "), "\n")[0]
		name = strings.Split(name, " (")[0]

		return switchVbranchCmd.RunE(cmd, []string{name})
	},
}

var materializeVbranchCmd = &cobra.Command{
	Use:     "materialize [name]",
	Short:   "Convert a virtual branch into a real Git branch",
	Aliases: []string{"m"},
	RunE: func(cmd *cobra.Command, args []string) error {
		var name string
		if len(args) > 0 {
			name = args[0]
		} else {
			// Interactive branch selection
			branches, err := vbranchManager.ListVirtualBranches()
			if err != nil {
				return fmt.Errorf("failed to list virtual branches: %w", err)
			}

			if len(branches) == 0 {
				return fmt.Errorf("no virtual branches to materialize")
			}

			options := make([]string, len(branches))
			for i, vb := range branches {
				changeCount := ui.Blue(fmt.Sprintf(" (%d changes)", len(vb.Changes)))
				options[i] = fmt.Sprintf("%s%s\n  Based on: %s", vb.Name, changeCount, vb.BaseBranch)
			}

			var selected string
			prompt := &survey.Select{
				Message: "Select branch to materialize:",
				Options: options,
			}
			if err := survey.AskOne(prompt, &selected); err != nil {
				return err
			}

			name = strings.Split(selected, "\n")[0]
			name = strings.Split(name, " (")[0]
		}

		// Confirm materialization
		vb, err := vbranchManager.GetVirtualBranch(name)
		if err != nil {
			return err
		}

		fmt.Printf("\nAbout to materialize virtual branch '%s'\n", name)
		fmt.Printf("This will:\n")
		fmt.Printf("1. Create a new Git branch '%s'\n", name)
		fmt.Printf("2. Apply %d changes\n", len(vb.Changes))
		if hasStash, _ := vbranchManager.HasStashedChanges(name); hasStash {
			fmt.Printf("3. Apply stashed changes\n")
		}
		fmt.Printf("4. Remove the virtual branch\n")

		if !ui.Confirm("Continue?") {
			ui.Info("Operation cancelled")
			return nil
		}

		if err := vbranchManager.MaterializeBranch(name); err != nil {
			return fmt.Errorf("failed to materialize branch: %w", err)
		}

		vbranchWatcher.SetActiveBranch("")
		ui.Successf("Successfully materialized '%s' into a real Git branch", name)

		// Offer to push the branch
		if ui.Confirm("Would you like to push this branch?") {
			gitService := git.NewShellGit()
			if err := gitService.Push(name, false); err != nil {
				return fmt.Errorf("failed to push branch: %w", err)
			}
			ui.Successf("Pushed branch '%s'", name)
		}

		return nil
	},
}

var statusVbranchCmd = &cobra.Command{
	Use:   "status",
	Short: "Show detailed status of virtual branches",
	RunE: func(cmd *cobra.Command, args []string) error {
		branches, err := vbranchManager.ListVirtualBranches()
		if err != nil {
			return fmt.Errorf("failed to list virtual branches: %w", err)
		}

		if len(branches) == 0 {
			ui.Info("No virtual branches found")
			return nil
		}

		// Show active branch first
		for _, vb := range branches {
			if vb.Active {
				ui.Successf("Active branch: %s", vb.Name)
				fmt.Printf("  Based on: %s\n", vb.BaseBranch)
				fmt.Printf("  Changes: %d files\n", len(vb.Changes))
				hasStash, _ := vbranchManager.HasStashedChanges(vb.Name)
				if hasStash {
					fmt.Printf("  Stashed changes: Yes\n")
				}
				fmt.Printf("  Last updated: %s\n", vb.LastUpdated.Format("2006-01-02 15:04:05"))
				fmt.Println()
				break
			}
		}

		// Then show other branches
		fmt.Println("Other branches:")
		for _, vb := range branches {
			if !vb.Active {
				hasStash, _ := vbranchManager.HasStashedChanges(vb.Name)
				stashStatus := ""
				if hasStash {
					stashStatus = " (has stashed changes)"
				}
				fmt.Printf("  %s: %d changes%s\n", vb.Name, len(vb.Changes), stashStatus)
			}
		}
		return nil
	},
}

var changesVbranchCmd = &cobra.Command{
	Use:   "changes [branch-name]",
	Short: "Show detailed changes in a virtual branch",
	RunE: func(cmd *cobra.Command, args []string) error {
		var branchName string
		if len(args) > 0 {
			branchName = args[0]
		} else {
			// Get active branch
			active, err := vbranchManager.GetActiveBranch()
			if err != nil {
				return fmt.Errorf("no active branch found")
			}
			branchName = active.Name
		}

		vb, err := vbranchManager.GetVirtualBranch(branchName)
		if err != nil {
			return fmt.Errorf("failed to get virtual branch: %w", err)
		}

		ui.Successf("Changes in branch: %s", vb.Name)
		fmt.Printf("Based on: %s\n\n", vb.BaseBranch)

		if len(vb.Changes) == 0 {
			ui.Info("No changes in this branch")
			return nil
		}

		// Group changes by type
		modified := []vbranch.Change{}
		added := []vbranch.Change{}
		deleted := []vbranch.Change{}
		renamed := []vbranch.Change{}

		for _, change := range vb.Changes {
			if strings.Contains(change.Diff, "rename from") {
				renamed = append(renamed, change)
			} else if strings.Contains(change.Diff, "new file") {
				added = append(added, change)
			} else if strings.Contains(change.Diff, "deleted file") {
				deleted = append(deleted, change)
			} else {
				modified = append(modified, change)
			}
		}

		// Show changes by type
		if len(added) > 0 {
			fmt.Printf("\n%s Added files:\n", ui.Green("+"))
			for _, change := range added {
				fmt.Printf("  %s\n", change.Path)
			}
		}

		if len(modified) > 0 {
			fmt.Printf("\n%s Modified files:\n", ui.Blue("~"))
			for _, change := range modified {
				fmt.Printf("  %s\n", change.Path)
			}
		}

		if len(deleted) > 0 {
			fmt.Printf("\n%s Deleted files:\n", ui.Red("-"))
			for _, change := range deleted {
				fmt.Printf("  %s\n", change.Path)
			}
		}

		if len(renamed) > 0 {
			fmt.Printf("\n%s Renamed files:\n", ui.Yellow("→"))
			for _, change := range renamed {
				fmt.Printf("  %s\n", change.Path)
			}
		}

		// Show stash status
		hasStash, _ := vbranchManager.HasStashedChanges(branchName)
		if hasStash {
			fmt.Printf("\n%s This branch has stashed changes\n", ui.Yellow("!"))
		}

		return nil
	},
}

var diffVbranchCmd = &cobra.Command{
	Use:   "diff [branch-name] [file-path]",
	Short: "Show the diff of changes in a virtual branch",
	RunE: func(cmd *cobra.Command, args []string) error {
		var branchName string
		var filePath string

		if len(args) > 0 {
			branchName = args[0]
			if len(args) > 1 {
				filePath = args[1]
			}
		} else {
			// Get active branch
			active, err := vbranchManager.GetActiveBranch()
			if err != nil {
				return fmt.Errorf("no active branch found")
			}
			branchName = active.Name
		}

		vb, err := vbranchManager.GetVirtualBranch(branchName)
		if err != nil {
			return fmt.Errorf("failed to get virtual branch: %w", err)
		}

		if len(vb.Changes) == 0 {
			ui.Info("No changes in this branch")
			return nil
		}

		// If a specific file is requested, show only its diff
		if filePath != "" {
			found := false
			for _, change := range vb.Changes {
				if change.Path == filePath {
					fmt.Println(change.Diff)
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("file %s not found in branch %s", filePath, branchName)
			}
			return nil
		}

		// Otherwise show all diffs
		ui.Successf("Changes in branch: %s", vb.Name)
		fmt.Printf("Based on: %s\n\n", vb.BaseBranch)

		for _, change := range vb.Changes {
			fmt.Printf("diff --git a/%s b/%s\n", change.Path, change.Path)
			fmt.Println(change.Diff)
			fmt.Println()
		}

		return nil
	},
}

// Quick create command for faster workflow
var newVbranchCmd = &cobra.Command{
	Use:     "new [description]",
	Short:   "Quickly create and switch to a new virtual branch",
	Aliases: []string{"n"},
	RunE: func(cmd *cobra.Command, args []string) error {
		var description string
		if len(args) > 0 {
			description = strings.Join(args, " ")
		} else {
			description = ui.AskString("What are you working on? ")
		}

		// Generate branch name from description
		name := generateBranchName(description)

		// Create the branch
		if err := createVbranchCmd.RunE(cmd, []string{name}); err != nil {
			return err
		}

		// Switch to it
		return switchVbranchCmd.RunE(cmd, []string{name})
	},
}

// Helper function to generate branch names
func generateBranchName(description string) string {
	// Convert to lowercase and replace spaces with hyphens
	name := strings.ToLower(description)
	name = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		if r == ' ' || r == '_' {
			return '-'
		}
		return -1
	}, name)

	// Ensure it starts with feature- if no prefix
	if !strings.HasPrefix(name, "feat-") && !strings.HasPrefix(name, "fix-") &&
		!strings.HasPrefix(name, "chore-") && !strings.HasPrefix(name, "docs-") {
		name = "feat-" + name
	}

	// Truncate if too long
	if len(name) > 50 {
		name = name[:50]
	}

	return name
}

func init() {
	rootCmd.AddCommand(vbranchCmd)
	vbranchCmd.AddCommand(createVbranchCmd)
	vbranchCmd.AddCommand(newVbranchCmd) // Add quick create command
	vbranchCmd.AddCommand(listVbranchCmd)
	vbranchCmd.AddCommand(switchVbranchCmd)
	vbranchCmd.AddCommand(focusVbranchCmd)
	vbranchCmd.AddCommand(materializeVbranchCmd)
	vbranchCmd.AddCommand(statusVbranchCmd)
	vbranchCmd.AddCommand(changesVbranchCmd)
	vbranchCmd.AddCommand(diffVbranchCmd)

	createVbranchCmd.Flags().StringP("base", "b", "", "Base branch to create from (defaults to current branch)")
}
