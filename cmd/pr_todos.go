package cmd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/gh"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var (
	sortBy   string
	showDiff bool
	showTime bool
)

var prTodosCmd = &cobra.Command{
	Use:   "todos [pr-num]",
	Short: "Show unresolved comment threads (uses current branch's PR if no number specified)",
	Long: `Display unresolved comment threads from a pull request in an organized view.
If no PR number is provided, it uses the PR associated with the current branch.

The output shows:
• File locations with line numbers and code context
• Comment threads with author, timestamp, and thread history
• Total count of unresolved threads and summary statistics

Flags:
  --sort    Sort threads by 'time' (newest first), 'file' (default), or 'count' (most discussed first)
  --diff    Show the code context around each comment
  --time    Show timestamps for each comment`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ghc := gh.NewClient()
		g := git.NewShellGit()

		var num int
		var err error

		if len(args) == 1 {
			// PR number provided
			num, err = strconv.Atoi(args[0])
			if err != nil {
				return err
			}
		} else {
			// Use current branch's PR
			branch, err := g.CurrentBranch()
			if err != nil {
				return err
			}

			// List PRs for this branch
			prs, err := ghc.ListPRs("open")
			if err != nil {
				return err
			}

			// Find PR for current branch
			var found bool
			for _, pr := range prs {
				if pr.Head.Ref == branch {
					num = pr.Number
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("no PR number provided and no PR found for current branch %q", branch)
			}
		}

		threads, err := app.ListUnresolvedThreads(ghc, num)
		if err != nil {
			return err
		}

		// Get PR details for context
		pr, err := app.GetPRDetails(ghc, num)
		if err != nil {
			return err
		}

		// Print header with PR context
		fmt.Printf("\n%s %s\n", ui.Sage("Pull Request"), ui.White(fmt.Sprintf("#%d: %s", num, pr.Title)))
		fmt.Printf("%s %s\n", ui.Sage("Branch:"), ui.White(fmt.Sprintf("%s → %s", pr.Head.Ref, pr.Base.Ref)))
		fmt.Printf("%s %s\n\n", ui.Sage("URL:"), ui.Blue(pr.HTMLURL))

		if len(threads) == 0 {
			fmt.Printf("%s %s\n", ui.Green("✓"), "No unresolved threads!")
			return nil
		}

		// Group and sort threads based on user preference
		switch sortBy {
		case "time":
			sort.Slice(threads, func(i, j int) bool {
				return len(threads[i].Comments) > 0 && len(threads[j].Comments) > 0 &&
					threads[i].Comments[len(threads[i].Comments)-1].Time.After(
						threads[j].Comments[len(threads[j].Comments)-1].Time)
			})
		case "count":
			sort.Slice(threads, func(i, j int) bool {
				return len(threads[i].Comments) > len(threads[j].Comments)
			})
		default: // "file"
			sort.Slice(threads, func(i, j int) bool {
				if threads[i].Path == threads[j].Path {
					return threads[i].Line < threads[j].Line
				}
				return threads[i].Path < threads[j].Path
			})
		}

		// Group threads by file
		fileThreads := make(map[string][]gh.UnresolvedThread)
		for _, t := range threads {
			fileThreads[t.Path] = append(fileThreads[t.Path], t)
		}

		// Calculate statistics
		totalComments := 0
		for _, t := range threads {
			totalComments += len(t.Comments)
		}

		// Print summary with detailed statistics
		fmt.Printf("%s Found %d unresolved thread(s) across %d file(s)\n",
			ui.Yellow("!"),
			len(threads),
			len(fileThreads))
		fmt.Printf("%s Total comments: %d (avg %.1f per thread)\n\n",
			ui.Yellow("→"),
			totalComments,
			float64(totalComments)/float64(len(threads)))

		// Print threads grouped by file
		for file, threads := range fileThreads {
			// Print file header with clickable link
			fileURL := fmt.Sprintf("%s/blob/%s/%s", strings.TrimSuffix(pr.HTMLURL, "/pull/"+strconv.Itoa(num)), pr.Head.Ref, file)
			fmt.Printf("%s %s\n%s %s\n",
				ui.Sage("File:"),
				ui.White(file),
				ui.Sage("Link:"),
				ui.Blue(fileURL))

			// Print each thread in the file
			for _, t := range threads {
				// Print location with clickable link
				lineURL := fmt.Sprintf("%s#L%d", fileURL, t.Line)
				fmt.Printf("\n  %s Line %s\n",
					ui.Yellow("→"),
					ui.Blue(fmt.Sprintf("%d (%s)", t.Line, lineURL)))

				if showDiff && t.CodeContext != "" {
					// Show code context with syntax highlighting
					fmt.Printf("    %s\n", ui.Gray("Code context:"))
					for _, line := range strings.Split(t.CodeContext, "\n") {
						fmt.Printf("      %s\n", ui.Gray(line))
					}
					fmt.Println()
				}

				// Print comments with proper indentation and timestamps
				for _, c := range t.Comments {
					// Format timestamp if enabled
					timeStr := ""
					if showTime && !c.Time.IsZero() {
						timeStr = fmt.Sprintf(" (%s)", c.Time.Local().Format("Jan 2, 15:04"))
					}

					// Indent and wrap comment body for better readability
					body := strings.ReplaceAll(c.Body, "\n", "\n        ")
					fmt.Printf("    %s%s\n      %s\n",
						ui.Blue("@"+c.User),
						ui.Gray(timeStr),
						body)
				}
			}
			fmt.Println()
		}

		// Print footer with helpful hints
		fmt.Printf("%s Use %s to view the full PR\n",
			ui.Sage("Tip:"),
			ui.White("sage pr status "+strconv.Itoa(num)))
		fmt.Printf("%s Use %s to check out this PR locally\n",
			ui.Sage("Tip:"),
			ui.White("sage pr checkout "+strconv.Itoa(num)))

		return nil
	},
}

func init() {
	prCmd.AddCommand(prTodosCmd)

	// Add flags for customizing the output
	prTodosCmd.Flags().StringVar(&sortBy, "sort", "file", "Sort threads by: 'time', 'file', or 'count'")
	prTodosCmd.Flags().BoolVar(&showDiff, "diff", false, "Show code context around comments")
	prTodosCmd.Flags().BoolVar(&showTime, "time", false, "Show timestamps for comments")
}
