package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/crazywolf132/sage/internal/gitutils"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

type commitInfo struct {
	Hash    string
	Author  string
	Date    time.Time
	Message string
	IsMerge bool
	Branch  string
	Tags    []string
	Stats   struct {
		Added    int
		Deleted  int
		Modified int
	}
}

var (
	showStats bool
	lastN     int
	showAll   bool
)

var historyCmd = &cobra.Command{
	Use:     "history [branch]",
	Aliases: []string{"log", "hist", "l"},
	Short:   "Show beautiful branch history",
	Long: `Display a detailed and beautiful history of a branch, including:
- Commit messages and authors
- Branch and merge information
- Tags and references
- File change statistics
- Time information

Examples:
  sage history              # Show history of current branch
  sage history main        # Show history of main branch
  sage history -n 10      # Show last 10 commits
  sage history --stats    # Show file change statistics
  sage history --all      # Show all commits including parent branches`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get target branch
		targetBranch := ""
		if len(args) > 0 {
			targetBranch = args[0]
		} else {
			var err error
			targetBranch, err = gitutils.GetCurrentBranch()
			if err != nil {
				return fmt.Errorf("failed to get current branch: %w", err)
			}
		}

		// Build git log command
		logFormat := "%H%x00%an%x00%at%x00%s%x00%P%x00%D"
		logCmd := []string{"log", "--format=" + logFormat}
		if lastN > 0 {
			logCmd = append(logCmd, "-n", fmt.Sprintf("%d", lastN))
		}
		if showStats {
			logCmd = append(logCmd, "--numstat")
		}

		if !showAll {
			// Get the current branch's upstream if it exists
			upstream, err := gitutils.DefaultRunner.RunGitCommandWithOutput("rev-parse", "--abbrev-ref", targetBranch+"@{upstream}")
			if err == nil {
				// Show only commits that are in the current branch but not in upstream
				logCmd = append(logCmd, fmt.Sprintf("%s..%s", strings.TrimSpace(upstream), targetBranch))
			} else {
				// If no upstream, just show the branch's commits
				logCmd = append(logCmd, targetBranch)
			}
		} else {
			logCmd = append(logCmd, targetBranch)
		}

		// Get commit history
		output, err := gitutils.DefaultRunner.RunGitCommandWithOutput(logCmd...)
		if err != nil {
			// If the command failed, try a simpler approach
			logCmd = []string{"log", "--format=" + logFormat}
			if lastN > 0 {
				logCmd = append(logCmd, "-n", fmt.Sprintf("%d", lastN))
			}
			if showStats {
				logCmd = append(logCmd, "--numstat")
			}
			logCmd = append(logCmd, targetBranch)
			output, err = gitutils.DefaultRunner.RunGitCommandWithOutput(logCmd...)
			if err != nil {
				return fmt.Errorf("failed to get git history: %w", err)
			}
		}

		// Parse commits
		commits := parseGitLog(output)

		// Print header
		fmt.Printf("\n%s %s\n\n",
			ui.ColoredText("Branch History:", ui.Bold+ui.Sage),
			ui.ColoredText(targetBranch, ui.Yellow))

		// Print commits in reverse order (latest at bottom)
		var lastDate string
		for i := len(commits) - 1; i >= 0; i-- {
			commit := commits[i]
			// Print date header if date changed
			currentDate := commit.Date.Format("Mon Jan 02 2006")
			if currentDate != lastDate {
				if lastDate != "" {
					fmt.Println()
				}
				fmt.Printf(" %s\n", ui.ColoredText(currentDate, ui.Bold+ui.Blue))
				lastDate = currentDate
			}

			// Print commit info
			printCommit(commit)
		}

		return nil
	},
}

func parseGitLog(output string) []commitInfo {
	var commits []commitInfo
	lines := strings.Split(strings.TrimSpace(output), "\n")

	var currentCommit *commitInfo
	var collectingStats bool

	for _, line := range lines {
		if strings.Contains(line, "\x00") {
			// This is a commit line
			if currentCommit != nil {
				commits = append(commits, *currentCommit)
			}

			parts := strings.Split(line, "\x00")
			timestamp, _ := strconv.ParseInt(parts[2], 10, 64)

			currentCommit = &commitInfo{
				Hash:    parts[0][:7], // Short hash
				Author:  parts[1],
				Date:    time.Unix(timestamp, 0),
				Message: parts[3],
				IsMerge: strings.HasPrefix(parts[3], "Merge"),
			}

			// Parse refs (branches and tags)
			if len(parts) > 5 && parts[5] != "" {
				refs := strings.Split(parts[5], ",")
				for _, ref := range refs {
					ref = strings.TrimSpace(ref)
					if strings.HasPrefix(ref, "tag: ") {
						currentCommit.Tags = append(currentCommit.Tags, strings.TrimPrefix(ref, "tag: "))
					} else if !strings.Contains(ref, "HEAD") {
						currentCommit.Branch = strings.TrimPrefix(ref, "origin/")
					}
				}
			}

			collectingStats = showStats
		} else if collectingStats && line != "" {
			// Parse numstat output
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				added, _ := strconv.Atoi(parts[0])
				deleted, _ := strconv.Atoi(parts[1])
				currentCommit.Stats.Added += added
				currentCommit.Stats.Deleted += deleted
				if added > 0 && deleted > 0 {
					currentCommit.Stats.Modified++
				}
			}
		}
	}

	if currentCommit != nil {
		commits = append(commits, *currentCommit)
	}

	return commits
}

func printCommit(commit commitInfo) {
	// Print commit hash and author
	fmt.Printf(" %s %s %s %s\n",
		ui.ColoredText("●", getCommitColor(commit)),
		ui.ColoredText(commit.Hash, ui.Yellow),
		ui.ColoredText("by", ui.Gray),
		ui.ColoredText("@"+strings.Split(commit.Author, " ")[0], ui.White))

	// Print commit message with indent
	messageLines := strings.Split(commit.Message, "\n")
	fmt.Printf("   %s\n", messageLines[0])
	for _, line := range messageLines[1:] {
		if strings.TrimSpace(line) != "" {
			fmt.Printf("   %s\n", ui.ColoredText(line, ui.Dim+ui.White))
		}
	}

	// Print additional info
	var info []string
	if commit.Branch != "" {
		info = append(info, ui.ColoredText(commit.Branch, ui.Blue))
	}
	for _, tag := range commit.Tags {
		info = append(info, ui.ColoredText("⚑ "+tag, ui.Purple))
	}
	if len(info) > 0 {
		fmt.Printf("   %s\n", strings.Join(info, " "))
	}

	// Print stats if available
	if showStats && (commit.Stats.Added > 0 || commit.Stats.Deleted > 0 || commit.Stats.Modified > 0) {
		fmt.Printf("   %s %s %s %s\n",
			ui.ColoredText("⟳", ui.Gray),
			ui.ColoredText(fmt.Sprintf("+%d", commit.Stats.Added), ui.Green),
			ui.ColoredText(fmt.Sprintf("-%d", commit.Stats.Deleted), ui.Red),
			ui.ColoredText(fmt.Sprintf("!%d", commit.Stats.Modified), ui.Yellow))
	}
}

func getCommitColor(commit commitInfo) string {
	if commit.IsMerge {
		return ui.Blue + ui.Bold
	}
	if len(commit.Tags) > 0 {
		return ui.Purple + ui.Bold
	}
	return ui.Sage
}

func init() {
	RootCmd.AddCommand(historyCmd)
	historyCmd.Flags().BoolVarP(&showStats, "stats", "s", false, "Show file change statistics")
	historyCmd.Flags().IntVarP(&lastN, "number", "n", 0, "Limit to last N commits")
	historyCmd.Flags().BoolVarP(&showAll, "all", "a", false, "Show all commits including parent branches")
}
