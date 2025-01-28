package cmd

import (
	"fmt"
	"strings"

	"github.com/crazywolf132/sage/internal/gitutils"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display repository status in a beautiful format",
	Long: `Display a detailed overview of your Git repository's current state.
	Shows information about:
	- Current branch
	- Uncommitted changes
	- Untracked files
	- Upstream status
	in a visually appealing format.`,
	RunE: runStatus,
}

func init() {
	RootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Check if we're in a git repository
	if err := gitutils.DefaultRunner.RunGitCommand("rev-parse", "--git-dir"); err != nil {
		return fmt.Errorf("not a git repository")
	}

	// Get current branch
	branch, err := gitutils.DefaultRunner.GetCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Get repository status
	status, err := gitutils.RunGitCommandWithOutput("status", "--porcelain")
	if err != nil {
		return fmt.Errorf("failed to get repository status: %w", err)
	}

	// Get upstream status
	upstream, _ := gitutils.RunGitCommandWithOutput("rev-list", "--count", "--left-right", "@{u}...HEAD")

	// Print status in beautiful format
	printStatus(branch, status, upstream)

	return nil
}

func printStatus(branch, status, upstream string) {
	// Print branch information
	fmt.Printf("\n%s %s\n", ui.ColoredText("⎇", ui.Blue), ui.ColoredText(branch, ui.Sage))

	// Parse and print status
	if status != "" {
		fmt.Println("\nChanges:")
		lines := strings.Split(strings.TrimSpace(status), "\n")
		for _, line := range lines {
			if len(line) < 4 {
				continue
			}
			statusCode := line[:2]
			filePath := line[3:]
			printStatusLine(statusCode, filePath)
		}
	} else {
		fmt.Println(ui.ColoredText("\n✓ Working tree clean", ui.Sage))
	}

	// Print upstream status if available
	if upstream != "" {
		parts := strings.Split(upstream, "\t")
		if len(parts) == 2 {
			behind := parts[0]
			ahead := parts[1]
			if behind != "0" {
				fmt.Printf("\n%s commits behind", behind)
			}
			if ahead != "0" {
				fmt.Printf("\n%s commits ahead", ahead)
			}
		}
	}
	fmt.Println()
}

func printStatusLine(statusCode, filePath string) {
	var symbol, color string

	switch statusCode {
	case " M": // Modified
		symbol = "📝"
		color = ui.Yellow
	case "M ": // Staged modified
		symbol = "✓"
		color = ui.Sage
	case "A ": // Added
		symbol = "➕"
		color = ui.Sage
	case " D": // Deleted
		symbol = "❌"
		color = ui.Red
	case "D ": // Staged deleted
		symbol = "🗑️"
		color = ui.Red
	case "??": // Untracked
		symbol = "❓"
		color = ui.Blue
	default:
		symbol = "•"
		color = ui.White
	}

	fmt.Printf("%s %s\n", symbol, ui.ColoredText(filePath, color))
}
