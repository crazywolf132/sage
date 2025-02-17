package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/crazywolf132/sage/internal/undo"
	"github.com/spf13/cobra"
)

var (
	undoID      string
	showHistory bool
)

var undoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Undo your last Git operation",
	Long: `Undo your last Git operation safely.

Just run 'sage undo' and we'll help you fix your last Git operation.
No need to remember complex Git commands - we'll handle it for you!`,
	Example: `  # Fix your last Git operation
  sage undo

  # See what you can undo
  sage undo --history`,
	RunE: func(cmd *cobra.Command, args []string) error {
		spinner := ui.NewSpinner()
		g := git.NewShellGit()
		s := undo.NewService(g)

		// Load history
		spinner.Start("Looking up your recent Git operations...")
		if err := s.LoadHistory("."); err != nil {
			spinner.StopFail()
			return fmt.Errorf("couldn't load Git history: %w", err)
		}
		spinner.StopSuccess()

		// Get operations
		ops := s.GetHistory().GetOperations("", time.Time{})
		if len(ops) == 0 {
			fmt.Printf("\n%s Nothing to undo yet\n", ui.Yellow("!"))
			fmt.Printf("\nTip: Use Sage commands like 'sage commit' or 'sage sync' first.\n")
			fmt.Printf("Then you can use 'sage undo' if something goes wrong.\n\n")
			return nil
		}

		// Just show history if requested
		if showHistory {
			showUndoHistory(ops)
			return nil
		}

		// Handle undo by ID
		if undoID != "" {
			return handleUndoByID(s, ops, undoID)
		}

		// Interactive mode - show last operation with clear preview
		return handleInteractiveUndo(s, ops)
	},
}

func init() {
	rootCmd.AddCommand(undoCmd)
	undoCmd.Flags().StringVarP(&undoID, "id", "i", "", "Undo a specific operation by ID")
	undoCmd.Flags().BoolVarP(&showHistory, "history", "H", false, "See what you can undo")
}

func handleInteractiveUndo(s *undo.Service, ops []undo.Operation) error {
	lastOp := ops[0] // Most recent operation

	// Header
	fmt.Printf("\n%s\n\n", ui.Bold("Let's undo your last Git operation"))

	// Show the operation details in a clean format
	fmt.Printf("%s %s\n", ui.Yellow("What happened:"), lastOp.Description)
	fmt.Printf("%s %s\n", ui.Yellow("When:"), formatTimestamp(lastOp.Timestamp))

	// Show context based on operation type
	switch lastOp.Category {
	case "commit":
		if len(lastOp.Metadata.Files) > 0 {
			fmt.Printf("%s %s\n", ui.Yellow("Changed files:"), strings.Join(lastOp.Metadata.Files, ", "))
		}
	case "merge", "rebase":
		if lastOp.Metadata.Branch != "" {
			fmt.Printf("%s %s\n", ui.Yellow("On branch:"), lastOp.Metadata.Branch)
		}
	}
	fmt.Println()

	// Show what will happen in a clear, action-oriented way
	fmt.Printf("%s\n", ui.Bold("Here's what we'll do:"))
	switch lastOp.Category {
	case "commit":
		fmt.Printf("1. Safely undo your last commit\n")
		fmt.Printf("2. Keep all your changes staged\n")
		fmt.Printf("3. Let you commit again when ready\n")
	case "merge":
		fmt.Printf("1. Cancel the problematic merge\n")
		fmt.Printf("2. Return your branch to its previous state\n")
		fmt.Printf("3. Clean up any temporary merge files\n")
	case "rebase":
		fmt.Printf("1. Stop the rebase operation\n")
		fmt.Printf("2. Put your branch back where it was\n")
		fmt.Printf("3. Preserve all your commits\n")
	case "branch":
		fmt.Printf("1. Restore your deleted branch\n")
		fmt.Printf("2. Recover all associated commits\n")
		fmt.Printf("3. Keep your current branch unchanged\n")
	}
	fmt.Println()

	// Safety check
	fmt.Printf("%s\n", ui.Bold("Safety Check:"))
	fmt.Printf("• Your files won't be lost\n")
	fmt.Printf("• Your other branches won't be affected\n")
	fmt.Printf("• You can redo this operation if needed\n")
	fmt.Println()

	// Ask for confirmation
	var proceed bool
	prompt := &survey.Confirm{
		Message: "Ready to undo?",
		Default: true,
	}
	if err := survey.AskOne(prompt, &proceed); err != nil {
		return err
	}
	if !proceed {
		fmt.Printf("\n%s Operation cancelled. Your repository is unchanged.\n\n", ui.Yellow("!"))
		return nil
	}

	// Show progress
	spinner := ui.NewSpinner()
	spinner.Start("Undoing last operation safely...")

	if err := s.UndoOperation(lastOp.ID); err != nil {
		spinner.StopFail()
		return fmt.Errorf("couldn't undo the operation: %v", err)
	}

	spinner.StopSuccess()

	// Show success and next steps
	fmt.Printf("\n%s Success! Here's what changed:\n", ui.Green("✓"))
	switch lastOp.Category {
	case "commit":
		fmt.Printf("• Your last commit was undone\n")
		fmt.Printf("• Your changes are ready to commit again\n")
		fmt.Printf("\n%s Try these commands:\n", ui.Bold("Next Steps"))
		fmt.Printf("• %s to see your changes\n", ui.Blue("git status"))
		fmt.Printf("• %s to commit when ready\n", ui.Blue("sage commit"))
	case "merge":
		fmt.Printf("• The merge was cancelled\n")
		fmt.Printf("• Your branch is back to normal\n")
		fmt.Printf("\n%s Try these commands:\n", ui.Bold("Next Steps"))
		fmt.Printf("• %s to see your branch status\n", ui.Blue("git status"))
		fmt.Printf("• %s to try merging again\n", ui.Blue("sage sync"))
	case "rebase":
		fmt.Printf("• The rebase was cancelled\n")
		fmt.Printf("• Your branch is back to its original state\n")
		fmt.Printf("\n%s Try these commands:\n", ui.Bold("Next Steps"))
		fmt.Printf("• %s to see your branch status\n", ui.Blue("git status"))
		fmt.Printf("• %s to update your branch differently\n", ui.Blue("sage sync"))
	case "branch":
		fmt.Printf("• Your branch was restored\n")
		fmt.Printf("• All commits were recovered\n")
		fmt.Printf("\n%s Try these commands:\n", ui.Bold("Next Steps"))
		fmt.Printf("• %s to switch to the restored branch\n", ui.Blue("sage switch <branch>"))
		fmt.Printf("• %s to see all branches\n", ui.Blue("git branch"))
	}
	fmt.Println()

	return nil
}

func handleUndoByID(s *undo.Service, ops []undo.Operation, id string) error {
	// Find the operation
	op := findOperationByID(ops, id)
	if op.ID == "" {
		fmt.Printf("\n%s Couldn't find that operation\n", ui.Yellow("!"))
		fmt.Printf("\nTip: Use 'sage undo --history' to see available operations\n")
		fmt.Printf("Or just run 'sage undo' to fix your last operation\n\n")
		return nil
	}

	// Show what we found
	fmt.Printf("\n%s Found this operation:\n\n", ui.Bold("Undo"))
	fmt.Printf("%s %s\n", ui.Yellow("What:"), op.Description)
	fmt.Printf("%s %s\n", ui.Yellow("When:"), formatTimestamp(op.Timestamp))
	if op.Metadata.Branch != "" {
		fmt.Printf("%s %s\n", ui.Yellow("Branch:"), op.Metadata.Branch)
	}
	fmt.Println()

	// Confirm
	var proceed bool
	prompt := &survey.Confirm{
		Message: "Undo this operation?",
		Default: true,
	}
	if err := survey.AskOne(prompt, &proceed); err != nil {
		return err
	}
	if !proceed {
		fmt.Printf("\n%s Operation cancelled. Your repository is unchanged.\n\n", ui.Yellow("!"))
		return nil
	}

	// Show progress
	spinner := ui.NewSpinner()
	spinner.Start("Undoing operation safely...")

	if err := s.UndoOperation(op.ID); err != nil {
		spinner.StopFail()
		return fmt.Errorf("couldn't undo the operation: %v", err)
	}

	spinner.StopSuccess()
	fmt.Printf("\n%s Operation undone successfully!\n", ui.Green("✓"))
	fmt.Printf("\nTip: Run 'git status' to see your repository's current state\n\n")
	return nil
}

func showUndoHistory(ops []undo.Operation) {
	fmt.Printf("\n%s\n\n", ui.Bold("Recent Git Operations"))

	for i, op := range ops {
		if i >= 5 { // Show only last 5 operations
			break
		}

		fmt.Printf("%s %s\n", ui.Yellow("•"), op.Description)
		fmt.Printf("  %s %s\n", ui.Gray("Type:"), strings.Title(op.Category))
		fmt.Printf("  %s %s\n", ui.Gray("When:"), formatTimestamp(op.Timestamp))
		fmt.Printf("  %s %s\n", ui.Gray("ID:"), op.ID[:8])

		// Show relevant context based on operation type
		switch op.Category {
		case "commit":
			if len(op.Metadata.Files) > 0 {
				files := op.Metadata.Files
				if len(files) > 3 {
					files = append(files[:2], "...")
				}
				fmt.Printf("  %s %s\n", ui.Gray("Files:"), strings.Join(files, ", "))
			}
		case "merge", "rebase":
			if op.Metadata.Branch != "" {
				fmt.Printf("  %s %s\n", ui.Gray("Branch:"), op.Metadata.Branch)
			}
		}
		fmt.Println()
	}

	if len(ops) > 5 {
		fmt.Printf("... and %d more operations\n", len(ops)-5)
	}

	fmt.Printf("\n%s\n", ui.Bold("How to undo:"))
	fmt.Printf("1. Just run: %s to undo your last operation\n", ui.Blue("sage undo"))
	fmt.Printf("2. Or undo a specific one: %s\n", ui.Blue("sage undo --id <ID>"))
	fmt.Println()
}

func formatTimestamp(t time.Time) string {
	duration := time.Since(t)
	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		return fmt.Sprintf("%d minute%s ago", mins, pluralize(mins))
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		return fmt.Sprintf("%d hour%s ago", hours, pluralize(hours))
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%d day%s ago", days, pluralize(days))
	default:
		return t.Format("Mon Jan 02 15:04:05")
	}
}

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func findOperationByID(ops []undo.Operation, id string) undo.Operation {
	for _, op := range ops {
		if strings.HasPrefix(op.ID, id) {
			return op
		}
	}
	return undo.Operation{}
}
