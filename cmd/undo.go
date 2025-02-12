package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/crazywolf132/sage/internal/undo"
	"github.com/spf13/cobra"
)

var (
	undoCount    int
	undoCategory string
	undoSince    string
	undoID       string
	showHistory  bool
	groupBy      string
	preview      bool
)

var undoCmd = &cobra.Command{
	Use:   "undo [count]",
	Short: "Undo Git operations with precision",
	Long: `Undo Git operations with detailed history tracking and selective undo capabilities.

Features:
• Interactive operation selection with preview
• Visual history with operation details
• Smart grouping by type, date, or branch
• Selective undo by operation ID
• Time-based filtering
• Automatic stash management

Examples:
  sage undo                      # Interactive undo with preview
  sage undo 3                    # Undo last 3 operations
  sage undo --id abc123          # Undo specific operation by ID
  sage undo --category commit    # Show only commit operations
  sage undo --since 24h         # Show operations from last 24 hours
  sage undo --group branch      # Group operations by branch
  sage undo --history          # Show detailed operation history`,
	RunE: func(cmd *cobra.Command, args []string) error {
		g := git.NewShellGit()
		s := undo.NewService(g)

		// Load existing history
		if err := s.LoadHistory("."); err != nil {
			return fmt.Errorf("failed to load undo history: %w", err)
		}

		// Parse time filter if provided
		var since time.Time
		if undoSince != "" {
			duration, err := time.ParseDuration(undoSince)
			if err != nil {
				return fmt.Errorf("invalid time duration: %w", err)
			}
			since = time.Now().Add(-duration)
		}

		// Get filtered operations
		ops := s.GetHistory().GetOperations(undoCategory, since)
		if len(ops) == 0 {
			fmt.Printf("\n%s No operations found matching the criteria\n\n", ui.Yellow("!"))
			fmt.Printf("Try:\n")
			fmt.Printf("• Removing filters (--category, --since)\n")
			fmt.Printf("• Using 'sage undo --history' to see all operations\n")
			fmt.Printf("• Checking if you're in the correct repository\n\n")
			return nil
		}

		// Just show history if requested
		if showHistory {
			printOperations(ops, groupBy)
			return nil
		}

		// Handle undo by ID
		if undoID != "" {
			if preview {
				if err := previewUndo(s, []undo.Operation{findOperationByID(ops, undoID)}); err != nil {
					return err
				}
			}
			return s.UndoOperation(undoID)
		}

		// Handle undo by count from args
		if len(args) > 0 {
			count, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid count: %w", err)
			}
			undoCount = count
		}

		// If count specified, undo that many operations
		if undoCount > 0 {
			if preview {
				if err := previewUndo(s, ops[:undoCount]); err != nil {
					return err
				}
			}
			return s.UndoLast(undoCount)
		}

		// Otherwise, show interactive selector
		return selectAndUndoOperation(s, ops)
	},
}

func init() {
	rootCmd.AddCommand(undoCmd)
	undoCmd.Flags().IntVarP(&undoCount, "count", "n", 0, "Number of operations to undo")
	undoCmd.Flags().StringVarP(&undoCategory, "category", "c", "", "Filter by category (commit, merge, rebase)")
	undoCmd.Flags().StringVarP(&undoSince, "since", "s", "", "Show operations since duration (e.g., 24h, 7d)")
	undoCmd.Flags().StringVarP(&undoID, "id", "i", "", "Undo specific operation by ID")
	undoCmd.Flags().BoolVarP(&showHistory, "history", "H", false, "Show operation history without undoing")
	undoCmd.Flags().StringVarP(&groupBy, "group", "g", "date", "Group operations by: date, type, branch")
	undoCmd.Flags().BoolVarP(&preview, "preview", "p", true, "Show preview of changes before undoing")
}

func findOperationByID(ops []undo.Operation, id string) undo.Operation {
	for _, op := range ops {
		if strings.HasPrefix(op.ID, id) {
			return op
		}
	}
	return undo.Operation{}
}

func printOperations(ops []undo.Operation, grouping string) {
	fmt.Printf("\n%s %s\n", ui.Bold(ui.Sage("Undo History")), ui.Gray(fmt.Sprintf("(%d operations)", len(ops))))

	switch grouping {
	case "type":
		printOperationsByType(ops)
	case "branch":
		printOperationsByBranch(ops)
	default:
		printOperationsByDate(ops)
	}
}

func printOperationsByDate(ops []undo.Operation) {
	var lastDate string
	for _, op := range ops {
		curDate := op.Timestamp.Format("Mon Jan 02 2006")
		if curDate != lastDate {
			if lastDate != "" {
				fmt.Println()
			}
			fmt.Printf("%s %s\n", ui.Blue("Date:"), ui.Bold(curDate))
			lastDate = curDate
		}
		printOperation(op)
	}
	fmt.Println()
}

func printOperationsByType(ops []undo.Operation) {
	typeGroups := make(map[string][]undo.Operation)
	for _, op := range ops {
		typeGroups[op.Category] = append(typeGroups[op.Category], op)
	}

	for _, category := range []string{"commit", "merge", "rebase", "stash"} {
		if ops, ok := typeGroups[category]; ok {
			fmt.Printf("\n%s %s\n", ui.Blue("Type:"), ui.Bold(strings.Title(category)))
			for _, op := range ops {
				printOperation(op)
			}
		}
	}
	fmt.Println()
}

func printOperationsByBranch(ops []undo.Operation) {
	branchGroups := make(map[string][]undo.Operation)
	for _, op := range ops {
		branch := op.Metadata.Branch
		if branch == "" {
			branch = "unknown"
		}
		branchGroups[branch] = append(branchGroups[branch], op)
	}

	for branch, ops := range branchGroups {
		fmt.Printf("\n%s %s\n", ui.Blue("Branch:"), ui.Bold(branch))
		for _, op := range ops {
			printOperation(op)
		}
	}
	fmt.Println()
}

func printOperation(op undo.Operation) {
	// Format operation line with appropriate symbol
	var symbol string
	switch op.Category {
	case "commit":
		symbol = "●"
	case "merge":
		symbol = "◆"
	case "rebase":
		symbol = "◇"
	case "stash":
		symbol = "⬡"
	default:
		symbol = "○"
	}

	fmt.Printf(" %s %s %s %s\n",
		ui.Sage(symbol),
		ui.Yellow(op.ID[:8]),
		ui.Gray(op.Timestamp.Format("15:04:05")),
		op.Description,
	)

	// Show metadata if available
	if op.Metadata.Branch != "" {
		fmt.Printf("   %s %s\n", ui.Gray("Branch:"), op.Metadata.Branch)
	}
	if len(op.Metadata.Files) > 0 {
		fileList := op.Metadata.Files
		if len(fileList) > 3 {
			fileList = append(fileList[:3], fmt.Sprintf("and %d more...", len(fileList)-3))
		}
		fmt.Printf("   %s %s\n", ui.Gray("Files:"), strings.Join(fileList, ", "))
	}
	if op.Metadata.Message != "" {
		msg := op.Metadata.Message
		if len(msg) > 80 {
			msg = msg[:77] + "..."
		}
		fmt.Printf("   %s %s\n", ui.Gray("Message:"), msg)
	}
}

func previewUndo(s *undo.Service, ops []undo.Operation) error {
	if len(ops) == 0 {
		return nil
	}

	fmt.Printf("\n%s The following operations will be undone:\n", ui.Bold(ui.Sage("Preview")))
	for i, op := range ops {
		fmt.Printf("\n%d. ", i+1)
		printOperation(op)
	}

	var confirm bool
	prompt := &survey.Confirm{
		Message: "Continue with undo?",
		Default: false,
	}
	if err := survey.AskOne(prompt, &confirm); err != nil {
		return fmt.Errorf("operation cancelled: %w", err)
	}

	if !confirm {
		return fmt.Errorf("operation cancelled by user")
	}

	return nil
}

func selectAndUndoOperation(s *undo.Service, ops []undo.Operation) error {
	// Create options for selection
	options := make([]string, len(ops))
	for i, op := range ops {
		var details string
		if op.Metadata.Branch != "" {
			details = fmt.Sprintf(" on %s", op.Metadata.Branch)
		}
		if len(op.Metadata.Files) > 0 {
			details += fmt.Sprintf(" (%d files)", len(op.Metadata.Files))
		}

		options[i] = fmt.Sprintf("[%s] %s: %s%s",
			op.ID[:8],
			op.Timestamp.Format("2006-01-02 15:04:05"),
			op.Description,
			details,
		)
	}

	// Show interactive selector
	var selected string
	prompt := &survey.Select{
		Message:  "Select operation to undo:",
		Options:  options,
		PageSize: 15,
	}
	if err := survey.AskOne(prompt, &selected); err != nil {
		return fmt.Errorf("operation cancelled: %w", err)
	}

	// Extract operation ID from selection
	opID := strings.Split(strings.Trim(strings.Split(selected, "]")[0], "["), " ")[0]

	// Show preview if enabled
	if preview {
		if err := previewUndo(s, []undo.Operation{findOperationByID(ops, opID)}); err != nil {
			return err
		}
	}

	return s.UndoOperation(opID)
}
