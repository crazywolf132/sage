package cmd

import (
	"fmt"
	"strconv"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/gh"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var prTodosCmd = &cobra.Command{
	Use:   "todos <pr-num>",
	Short: "Show unresolved comment threads",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		num, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		ghc := gh.NewClient()
		threads, err := app.ListUnresolvedThreads(ghc, num)
		if err != nil {
			return err
		}
		if len(threads) == 0 {
			fmt.Println(ui.Green("No unresolved threads!"))
			return nil
		}
		for _, t := range threads {
			fmt.Printf("\n%s File: %s, Line: %d\n", ui.Yellow("→"), t.Path, t.Line)
			for _, c := range t.Comments {
				fmt.Printf("   @%s: %s\n", c.User, c.Body)
			}
		}
		return nil
	},
}

func init() {
	prCmd.AddCommand(prTodosCmd)
}
