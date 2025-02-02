package cmd

import (
	"fmt"
	"strconv"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/gh"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var prCloseCmd = &cobra.Command{
	Use:   "close <pr-num>",
	Short: "Close a PR without merging",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		num, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		ghc := gh.NewClient()
		if err := app.ClosePR(ghc, num); err != nil {
			return err
		}
		fmt.Printf("%s Closed PR #%d\n", ui.Green("✓"), num)
		return nil
	},
}

func init() {
	prCmd.AddCommand(prCloseCmd)
}
