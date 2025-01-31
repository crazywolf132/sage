package cmd

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/gh"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var listState string

var prListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pull requests",
	RunE: func(cmd *cobra.Command, args []string) error {
		ghc := gh.NewClient()
		if listState == "" {
			listState = "open"
		}
		prs, err := app.ListPRs(ghc, listState)
		if err != nil {
			return err
		}
		for _, pr := range prs {
			fmt.Printf("%s #%d [%s] %s\n", ui.Sage("•"), pr.Number, pr.State, pr.Title)
		}
		return nil
	},
}

func init() {
	prCmd.AddCommand(prListCmd)
	prListCmd.Flags().StringVar(&listState, "state", "open", "PRs by state (open, closed, all)")
}
