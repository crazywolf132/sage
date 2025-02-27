package cmd

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var (
	stageAll      bool
	stagePatterns []string
	stageAI       bool
)

var stageCmd = &cobra.Command{
	Use:   "stage [patterns...]",
	Short: "Stage files for commit",
	Long: `Stage files for commit. Without arguments, shows an interactive file selector.
With patterns, stages all files matching the glob patterns.

After staging files, use 'sage commit --only-staged' to commit only the staged changes.

Examples:
  # Interactive mode
  sage stage

  # AI-powered grouping
  sage stage --ai

  # Stage specific files by pattern
  sage stage "*.go" "cmd/*.go"

  # Stage everything
  sage stage -a
  sage stage --all`,
	RunE: func(cmd *cobra.Command, args []string) error {
		g := git.NewShellGit()

		// If --all flag is provided, stage everything
		if stageAll {
			return g.StageAll()
		}

		// If patterns are provided as arguments, use them
		if len(args) > 0 {
			stagePatterns = args
		}

		err := app.StageFiles(g, stagePatterns, stageAI)
		if err == nil {
			// If staging was successful, remind about committing with --only-staged
			fmt.Printf("\nTip: Use %s to commit only these staged changes\n", ui.Blue("sage commit --only-staged"))
		}
		return err
	},
}

func init() {
	rootCmd.AddCommand(stageCmd)
	stageCmd.Flags().BoolVarP(&stageAll, "all", "a", false, "Stage all changes")
	stageCmd.Flags().StringSliceVarP(&stagePatterns, "pattern", "p", nil, "Glob patterns to match files to stage")
	stageCmd.Flags().BoolVar(&stageAI, "ai", false, "Use AI to group changes by functionality")
}
