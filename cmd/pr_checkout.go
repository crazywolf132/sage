package cmd

import (
	"fmt"
	"strconv"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/gh"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var prCheckoutCmd = &cobra.Command{
	Use:   "checkout <pr-num>",
	Short: "Check out PR locally",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		num, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		g := git.NewShellGit()
		ghc := gh.NewClient()
		branch, err := app.CheckoutPR(g, ghc, num)
		if err != nil {
			return err
		}
		fmt.Printf("%s Checked out PR #%d to branch %q\n", ui.Green("✓"), num, branch)
		return nil
	},
}

func init() {
	prCmd.AddCommand(prCheckoutCmd)
}
