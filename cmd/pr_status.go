package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/crazywolf132/sage/internal/githubutils"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var prStatusCmd = &cobra.Command{
	Use:   "status [pr-number]",
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
		// 1. Get token
		token, err := githubutils.GetGitHubToken()
		if err != nil {
			return err
		}
		if token == "" {
			return errors.New("no GitHub token found; install GH CLI or set SAGE_GITHUB_TOKEN / GITHUB_TOKEN")
		}

		// 2. Get owner/repo
		owner, repo, err := githubutils.FindRepoOwnerAndName()
		if err != nil {
			return err
		}

		var prNumber int
		if len(args) == 1 {
			// If PR number provided, use it
			prNumber, err = strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[0])
			}
		} else {
			// If no PR number, try to get current branch's PR
			currentPR, err := githubutils.GetCurrentBranchPR(token, owner, repo)
			if err != nil {
				return fmt.Errorf("failed to get PR for current branch: %w", err)
			}
			if currentPR == nil {
				return errors.New("no pull request found for current branch")
			}
			prNumber = currentPR.Number
		}

		// 3. Get PR details
		pr, err := githubutils.GetPullRequestDetails(token, owner, repo, prNumber)
		if err != nil {
			return err
		}

		// 4. Print PR information in a beautiful format
		printPRStatus(pr)
		return nil
	},
}

func printPRStatus(pr *githubutils.PullRequestDetails) {
	// Title section
	fmt.Printf("\n%s #%d: %s\n", ui.ColoredText("Pull Request", ui.Sage), pr.Number, ui.ColoredText(pr.Title, ui.White))
	fmt.Printf("%s\n\n", ui.ColoredText(pr.HTMLURL, ui.Blue))

	// Status indicators
	statusColor := ui.Yellow
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

	fmt.Printf("Status: %s\n", ui.ColoredText(strings.ToUpper(status), statusColor))

	// Branch information
	fmt.Printf("Branch: %s → %s\n", ui.ColoredText(pr.Head.Ref, ui.Yellow), ui.ColoredText(pr.Base.Ref, ui.Yellow))

	// Review status
	if len(pr.Reviews) > 0 {
		fmt.Printf("\n%s\n", ui.ColoredText("Reviews:", ui.Sage))
		for _, review := range pr.Reviews {
			reviewColor := ui.White
			switch review.State {
			case "APPROVED":
				reviewColor = ui.Sage
			case "CHANGES_REQUESTED":
				reviewColor = ui.Red
			case "COMMENTED":
				reviewColor = ui.Blue
			}
			fmt.Printf("  %s by @%s\n", ui.ColoredText(review.State, reviewColor), review.User.Login)
		}
	}

	// CI Status
	if len(pr.Checks) > 0 {
		fmt.Printf("\n%s\n", ui.ColoredText("Checks:", ui.Sage))
		for _, check := range pr.Checks {
			checkColor := ui.White
			switch check.Status {
			case "success":
				checkColor = ui.Sage
			case "failure":
				checkColor = ui.Red
			case "pending":
				checkColor = ui.Yellow
			}
			fmt.Printf("  %s: %s\n", check.Name, ui.ColoredText(check.Status, checkColor))
		}
	}

	// Description
	if pr.Body != "" {
		fmt.Printf("\n%s\n", ui.ColoredText("Description:", ui.Sage))
		fmt.Printf("%s\n", pr.Body)
	}

	// Timeline (recent commits)
	if len(pr.Timeline) > 0 {
		fmt.Printf("\n%s\n", ui.ColoredText("Recent Commits:", ui.Sage))
		for _, event := range pr.Timeline {
			if event.Event == "committed" {
				timestamp := event.CreatedAt.Format(time.RFC822)
				// Get first line of commit message
				message := strings.Split(event.Message, "\n")[0]
				fmt.Printf("  %s: %s (%s) by @%s\n",
					ui.ColoredText(timestamp, ui.White),
					message,
					ui.ColoredText(event.SHA, ui.Yellow),
					event.Actor.Login)
			}
		}
	}
}

func init() {
	prCmd.AddCommand(prStatusCmd)
}
