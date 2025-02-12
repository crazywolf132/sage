package cmd

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/sage/internal/ai"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var smartCreateCmd = &cobra.Command{
	Use:     "new [description]",
	Short:   "Create a new virtual branch with AI-generated name",
	Example: "sage vbranch new \"Add user authentication with OAuth\"",
	RunE: func(cmd *cobra.Command, args []string) error {
		var description string
		if len(args) > 0 {
			description = strings.Join(args, " ")
		} else {
			prompt := &survey.Input{
				Message: "What are you working on?",
			}
			survey.AskOne(prompt, &description)
		}

		// Use AI to generate a branch name
		llm := ai.NewOpenAILLM()
		aiClient := ai.NewClient(llm)
		branchName, err := aiClient.GenerateBranchName(description)
		if err != nil {
			return fmt.Errorf("failed to generate branch name: %w", err)
		}

		// Confirm branch name with user
		ui.Infof("Generated branch name: %s", branchName)
		if !ui.Confirm("Would you like to use this branch name?") {
			prompt := &survey.Input{
				Message: "Enter your preferred branch name:",
				Default: branchName,
			}
			survey.AskOne(prompt, &branchName)
		}

		// Create the virtual branch
		gitService := git.NewShellGit()
		currentBranch, err := gitService.CurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}

		vb, err := vbranchManager.CreateVirtualBranch(branchName, currentBranch)
		if err != nil {
			return fmt.Errorf("failed to create virtual branch: %w", err)
		}

		vbranchWatcher.SetActiveBranch(branchName)
		ui.Successf("Created and switched to virtual branch '%s'", vb.Name)
		return nil
	},
}

var smartSwitchCmd = &cobra.Command{
	Use:   "focus",
	Short: "Interactively switch between virtual branches",
	RunE: func(cmd *cobra.Command, args []string) error {
		branches, err := vbranchManager.ListVirtualBranches()
		if err != nil {
			return fmt.Errorf("failed to list virtual branches: %w", err)
		}

		if len(branches) == 0 {
			ui.Info("No virtual branches found")
			if ui.Confirm("Would you like to create one?") {
				return smartCreateCmd.RunE(cmd, args)
			}
			return nil
		}

		// Build branch options
		options := make([]string, len(branches))
		for i, vb := range branches {
			status := " "
			if vb.Active {
				status = "*"
			}
			options[i] = fmt.Sprintf("%s %s (%d changes)", status, vb.Name, len(vb.Changes))
		}

		var selected string
		prompt := &survey.Select{
			Message: "Select a virtual branch to focus on:",
			Options: options,
		}
		survey.AskOne(prompt, &selected)

		// Extract branch name from selection
		branchName := strings.Split(strings.TrimSpace(selected), " ")[1]

		if err := vbranchManager.ApplyVirtualBranch(branchName); err != nil {
			return fmt.Errorf("failed to switch to virtual branch: %w", err)
		}

		vbranchWatcher.SetActiveBranch(branchName)
		ui.Successf("Switched to virtual branch '%s'", branchName)
		return nil
	},
}

var moveChangesCmd = &cobra.Command{
	Use:   "move",
	Short: "Move changes between virtual branches",
	RunE: func(cmd *cobra.Command, args []string) error {
		branches, err := vbranchManager.ListVirtualBranches()
		if err != nil {
			return fmt.Errorf("failed to list virtual branches: %w", err)
		}

		if len(branches) < 2 {
			return fmt.Errorf("need at least two virtual branches to move changes")
		}

		// Select source branch
		var sourceBranch string
		{
			options := make([]string, len(branches))
			for i, vb := range branches {
				options[i] = fmt.Sprintf("%s (%d changes)", vb.Name, len(vb.Changes))
			}

			prompt := &survey.Select{
				Message: "Select source branch:",
				Options: options,
			}
			survey.AskOne(prompt, &sourceBranch)
			sourceBranch = strings.Split(sourceBranch, " ")[0]
		}

		// Get changes from source branch
		vb, err := vbranchManager.GetVirtualBranch(sourceBranch)
		if err != nil {
			return err
		}

		if len(vb.Changes) == 0 {
			return fmt.Errorf("no changes in source branch")
		}

		// Select changes to move
		var selectedChanges []string
		{
			options := make([]string, len(vb.Changes))
			for i, change := range vb.Changes {
				options[i] = change.Path
			}

			prompt := &survey.MultiSelect{
				Message: "Select changes to move:",
				Options: options,
			}
			survey.AskOne(prompt, &selectedChanges)
		}

		if len(selectedChanges) == 0 {
			return fmt.Errorf("no changes selected")
		}

		// Select target branch
		var targetBranch string
		{
			options := make([]string, 0, len(branches)-1)
			for _, vb := range branches {
				if vb.Name != sourceBranch {
					options = append(options, vb.Name)
				}
			}

			prompt := &survey.Select{
				Message: "Select target branch:",
				Options: options,
			}
			survey.AskOne(prompt, &targetBranch)
		}

		// Move the changes
		if err := vbranchManager.MoveChanges(sourceBranch, targetBranch, selectedChanges); err != nil {
			return fmt.Errorf("failed to move changes: %w", err)
		}

		ui.Successf("Successfully moved %d changes from '%s' to '%s'", len(selectedChanges), sourceBranch, targetBranch)
		return nil
	},
}

var publishCmd = &cobra.Command{
	Use:   "publish [name]",
	Short: "Convert a virtual branch into a real branch and push it",
	RunE: func(cmd *cobra.Command, args []string) error {
		var branchName string
		if len(args) > 0 {
			branchName = args[0]
		} else {
			branches, err := vbranchManager.ListVirtualBranches()
			if err != nil {
				return fmt.Errorf("failed to list virtual branches: %w", err)
			}

			options := make([]string, len(branches))
			for i, vb := range branches {
				options[i] = fmt.Sprintf("%s (%d changes)", vb.Name, len(vb.Changes))
			}

			var selected string
			prompt := &survey.Select{
				Message: "Select a virtual branch to publish:",
				Options: options,
			}
			survey.AskOne(prompt, &selected)
			branchName = strings.Split(selected, " ")[0]
		}

		// Ask for commit message
		var commitMsg string
		prompt := &survey.Input{
			Message: "Enter commit message:",
			Default: fmt.Sprintf("Feature: %s", branchName),
		}
		survey.AskOne(prompt, &commitMsg)

		// Materialize the branch
		if err := vbranchManager.MaterializeBranch(branchName); err != nil {
			return fmt.Errorf("failed to materialize branch: %w", err)
		}

		// Push the branch
		gitService := git.NewShellGit()
		if err := gitService.Push(branchName, false); err != nil {
			return fmt.Errorf("failed to push branch: %w", err)
		}

		ui.Successf("Successfully published virtual branch '%s' as a real branch", branchName)
		return nil
	},
}

func init() {
	vbranchCmd.AddCommand(smartCreateCmd)
	vbranchCmd.AddCommand(smartSwitchCmd)
	vbranchCmd.AddCommand(moveChangesCmd)
	vbranchCmd.AddCommand(publishCmd)
}
