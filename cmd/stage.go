package cmd

import (
	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/git"
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

		return app.StageFiles(g, stagePatterns, stageAI)
	},
}

func init() {
	rootCmd.AddCommand(stageCmd)
	stageCmd.Flags().BoolVarP(&stageAll, "all", "a", false, "Stage all changes")
	stageCmd.Flags().StringSliceVarP(&stagePatterns, "pattern", "p", nil, "Glob patterns to match files to stage")
	stageCmd.Flags().BoolVar(&stageAI, "ai", false, "Use AI to group changes by functionality")
}
