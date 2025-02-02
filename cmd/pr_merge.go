package cmd

import (
	"fmt"
	"strconv"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/gh"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var method string

var prMergeCmd = &cobra.Command{
	Use:   "merge <pr-num>",
	Short: "Merge a pull request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		num, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		if method == "" {
			method = "merge"
		}
		ghc := gh.NewClient()
		if err := app.MergePR(ghc, num, method); err != nil {
			return err
		}
		fmt.Printf("%s Merged PR #%d with method=%s\n", ui.Green("✓"), num, method)
		return nil
	},
}

func init() {
	prCmd.AddCommand(prMergeCmd)
	prMergeCmd.Flags().StringVar(&method, "method", "merge", "merge|squash|rebase")
}
