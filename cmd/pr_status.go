package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/gh"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var prStatusCmd = &cobra.Command{
	Use:   "status [pr-num]",
	Short: "Show detailed status of a pull request",
	Long: `Display comprehensive information about a pull request including:
- Title and description
- Current status (open/closed/merged)
- Review status
- CI/CD checks status
- Branch information
- Timeline of events`,
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

		details, err := app.GetPRDetails(ghc, num)
		if err != nil {
			return err
		}

		printPRStatus(details)
		return nil
	},
}

func printPRStatus(pr *gh.PullRequest) {
	// Title section
	fmt.Printf("\n%s #%d: %s\n", ui.Sage("Pull Request"), pr.Number, ui.White(pr.Title))
	fmt.Printf("%s\n\n", ui.Blue(pr.HTMLURL))

	// Status indicators
	var statusColor func(string) string = ui.Yellow
	if pr.State == "closed" {
		if pr.Merged {
			statusColor = ui.Blue
		} else {
			statusColor = ui.Red
		}
	} else if pr.Draft {
		statusColor = ui.White
	} else if pr.State == "open" {
		statusColor = ui.Sage
	}

	status := pr.State
	if pr.Draft {
		status = "draft"
	}
	if pr.Merged {
		status = "merged"
	}

	fmt.Printf("Status: %s\n", statusColor(strings.ToUpper(status)))

	// Branch information
	fmt.Printf("Branch: %s → %s\n", ui.Yellow(pr.Head.Ref), ui.Yellow(pr.Base.Ref))

	// Review status
	if len(pr.Reviews) > 0 {
		fmt.Printf("\n%s\n", ui.Sage("Reviews:"))
		for _, review := range pr.Reviews {
			var reviewColor func(string) string = ui.White
			switch review.State {
			case "APPROVED":
				reviewColor = ui.Sage
			case "CHANGES_REQUESTED":
				reviewColor = ui.Red
			case "COMMENTED":
				reviewColor = ui.Blue
			}
			fmt.Printf("  %s by @%s\n", reviewColor(review.State), review.User.Login)
		}
	}

	// CI Status
	if len(pr.Checks) > 0 {
		fmt.Printf("\n%s\n", ui.Sage("Checks:"))
		for _, check := range pr.Checks {
			var checkColor func(string) string = ui.White
			switch check.Status {
			case "success":
				checkColor = ui.Sage
			case "failure":
				checkColor = ui.Red
			case "pending":
				checkColor = ui.Yellow
			}
			fmt.Printf("  %s: %s\n", check.Name, checkColor(check.Status))
		}
	}

	// Description
	if pr.Body != "" {
		fmt.Printf("\n%s\n", ui.Sage("Description:"))
		fmt.Printf("%s\n", pr.Body)
	}

	// Timeline (recent commits)
	if len(pr.Timeline) > 0 {
		fmt.Printf("\n%s\n", ui.Sage("Recent Commits:"))
		for _, event := range pr.Timeline {
			if event.Event == "committed" {
				// Convert UTC time to local timezone
				localTime := event.CreatedAt.Local()
				timestamp := localTime.Format(time.RFC822)
				// Get first line of commit message
				message := strings.Split(event.Message, "\n")[0]
				fmt.Printf("  %s: %s (%s) by @%s\n",
					ui.White(timestamp),
					message,
					ui.Yellow(event.SHA[:7]),
					event.Actor.Login)
			}
		}
	}
}

func init() {
	prCmd.AddCommand(prStatusCmd)
}
