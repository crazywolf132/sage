package cmd

import (
	"fmt"
	"strings"

	"github.com/crazywolf132/sage/internal/gitutils"
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
	ahead, behind := 0, 0
	if upstream, err := gitutils.RunGitCommandWithOutput("rev-list", "--left-right", "--count", "@{u}...HEAD"); err == nil {
		fmt.Sscanf(upstream, "%d\t%d", &behind, &ahead)
	}

	// Print status in beautiful format
	fmt.Printf("\n📊 Repository Status\n")
	fmt.Printf("   ⎇  %s\n", branch)

	if ahead > 0 || behind > 0 {
		if ahead > 0 {
			fmt.Printf("   ⬆️  %d commit(s) ahead\n", ahead)
		}
		if behind > 0 {
			fmt.Printf("   ⬇️  %d commit(s) behind\n", behind)
		}
	}

	// Parse and print status
	if status != "" {
		fmt.Println("\n   Changes:")
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
		fmt.Println("\n   ✨ Working tree clean")
	}

	fmt.Println() // Add final newline
	return nil
}

func printStatusLine(statusCode, filePath string) {
	var symbol, description string

	switch statusCode {
	case "M ", " M":
		symbol = "📝"
		description = "Modified"
	case "A ", "AM":
		symbol = "✨"
		description = "Added"
	case "D ", " D":
		symbol = "🗑️ "
		description = "Deleted"
	case "R ":
		symbol = "📋"
		description = "Renamed"
	case "C ":
		symbol = "📑"
		description = "Copied"
	case "??":
		symbol = "❓"
		description = "Untracked"
	case "UU":
		symbol = "⚠️ "
		description = "Conflict"
	default:
		symbol = "  "
		description = "Unknown"
	}

	fmt.Printf("      %s %s (%s)\n", symbol, filePath, description)
}
