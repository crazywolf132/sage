package cmd

import (
	"fmt"
	"strings"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show repository status",
	RunE: func(cmd *cobra.Command, args []string) error {
		g := git.NewShellGit()
		st, err := app.GetRepoStatus(g)
		if err != nil {
			return err
		}

		// Header with branch info
		fmt.Printf("\n%s Repository Status\n", ui.Bold(ui.Sage("📊")))
		fmt.Printf("\n%s %s\n", ui.Bold("Branch:"), ui.Yellow(st.Branch))

		// Clean state
		if len(st.Changes) == 0 {
			fmt.Printf("\n%s %s\n\n", ui.Green("✓"), ui.Bold("Working directory is clean"))
			return nil
		}

		// Group changes by type
		staged := make([]app.FileChange, 0)
		unstaged := make([]app.FileChange, 0)
		untracked := make([]app.FileChange, 0)

		for _, c := range st.Changes {
			switch c.Symbol {
			case "?":
				untracked = append(untracked, c)
			case "A", "M", "D", "R":
				if strings.HasPrefix(c.Description, "Staged") {
					staged = append(staged, c)
				} else {
					unstaged = append(unstaged, c)
				}
			}
		}

		// Print changes by section
		if len(staged) > 0 {
			fmt.Printf("\n%s\n", ui.Bold(ui.Sage("Staged Changes:")))
			for _, c := range staged {
				symbol := getSymbolEmoji(c.Symbol)
				fmt.Printf("  %s %s\n", symbol, ui.White(c.File))
			}
		}

		if len(unstaged) > 0 {
			fmt.Printf("\n%s\n", ui.Bold(ui.Yellow("Changes not staged:")))
			for _, c := range unstaged {
				symbol := getSymbolEmoji(c.Symbol)
				fmt.Printf("  %s %s\n", symbol, ui.White(c.File))
			}
		}

		if len(untracked) > 0 {
			fmt.Printf("\n%s\n", ui.Bold(ui.Blue("Untracked files:")))
			for _, c := range untracked {
				fmt.Printf("  %s %s\n", "📄", ui.White(c.File))
			}
		}

		fmt.Println()
		return nil
	},
}

func getSymbolEmoji(symbol string) string {
	switch symbol {
	case "M":
		return "📝"
	case "A":
		return "✨"
	case "D":
		return "🗑️"
	case "R":
		return "📋"
	default:
		return "•"
	}
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
