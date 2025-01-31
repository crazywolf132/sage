package cmd

import (
	"fmt"
	"strconv"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/gh"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var prStatusCmd = &cobra.Command{
	Use:   "status <pr-num>",
	Short: "Show PR status details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		num, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		ghc := gh.NewClient()
		details, err := app.GetPRDetails(ghc, num)
		if err != nil {
			return err
		}
		fmt.Printf("%s PR #%d: %s\n", ui.Sage("ℹ"), details.Number, details.Title)
		fmt.Printf("   URL: %s\n   State: %s\n", details.HTMLURL, details.State)
		return nil
	},
}

func init() {
	prCmd.AddCommand(prStatusCmd)
}
