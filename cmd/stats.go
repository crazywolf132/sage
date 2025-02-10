package cmd

import (
	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/spf13/cobra"
)

var (
	statsTimeRange string
	statsLimit     int
	statsDetailed  bool
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show repository statistics and analytics",
	Long: `Display comprehensive Git repository statistics and analytics including:
- Commit activity and trends
- File change frequency
- Author contributions
- Branch activity
- Common merge conflict areas
- Performance metrics

Examples:
  # Show basic repository stats
  sage stats

  # Show detailed stats for last month
  sage stats --range month --detailed

  # Show top 10 most active files
  sage stats --limit 10`,
	RunE: func(cmd *cobra.Command, args []string) error {
		g := git.NewShellGit()
		return app.GetStats(g, app.StatsOptions{
			TimeRange: statsTimeRange,
			Limit:     statsLimit,
			Detailed:  statsDetailed,
		})
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.Flags().StringVarP(&statsTimeRange, "range", "r", "all", "Time range for stats (day|week|month|year|all)")
	statsCmd.Flags().IntVarP(&statsLimit, "limit", "n", 10, "Limit the number of items in each category")
	statsCmd.Flags().BoolVarP(&statsDetailed, "detailed", "d", false, "Show detailed statistics")
}
