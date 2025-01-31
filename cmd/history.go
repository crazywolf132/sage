package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
)

var (
	historyLimit int
	showStats    bool
	showAll      bool
)

var historyCmd = &cobra.Command{
	Use:   "history [branch]",
	Short: "Show a beautiful commit history",
	Long: `Displays a formatted log of commits on the current or specified branch.
You can limit the number of commits, show stats, etc.`,
	Args:    cobra.MaximumNArgs(1),
	Aliases: []string{"hist", "log", "l"},
	RunE: func(cmd *cobra.Command, args []string) error {
		g := git.NewShellGit()

		branch := ""
		if len(args) == 1 {
			branch = args[0]
		}
		hist, err := app.GetHistory(g, branch, historyLimit, showStats, showAll)
		if err != nil {
			return err
		}

		fmt.Printf("\n%s %s\n", ui.Bold(ui.Sage("Branch History:")), ui.Yellow(hist.BranchName))
		if len(hist.Commits) == 0 {
			fmt.Println(ui.Green("No commits found."))
			return nil
		}

		// We'll group by date for a nice display
		var lastDate string
		for i := len(hist.Commits) - 1; i >= 0; i-- {
			c := hist.Commits[i]
			curDate := c.Date.Format("Mon Jan 02 2006")
			if curDate != lastDate {
				if lastDate != "" {
					fmt.Println()
				}
				fmt.Printf("%s %s\n", ui.Blue("Date:"), ui.Bold(curDate))
				lastDate = curDate
			}

			// Print commit line
			fmt.Printf(" %s %s %s @%s\n",
				ui.Sage("●"),
				ui.Yellow(c.ShortHash),
				ui.Gray("by"),
				ui.White(strings.Split(c.AuthorName, " ")[0]),
			)
			fmt.Printf("   %s\n", c.Message)

			if showStats && (c.Stats.Added > 0 || c.Stats.Deleted > 0 || c.Stats.Modified > 0) {
				fmt.Printf("   %s +%d -%d ~%d\n", ui.Gray("Stats:"), c.Stats.Added, c.Stats.Deleted, c.Stats.Modified)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(historyCmd)
	historyCmd.Flags().IntVarP(&historyLimit, "number", "n", 0, "Limit to last N commits")
	historyCmd.Flags().BoolVarP(&showStats, "stats", "s", false, "Show file change statistics")
	historyCmd.Flags().BoolVarP(&showAll, "all", "a", false, "Show all commits including merges from other branches")
}
