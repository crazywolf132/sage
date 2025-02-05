package app

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/ai"
	"github.com/crazywolf132/sage/internal/gh"
	"github.com/crazywolf132/sage/internal/git"
)

type PRUpdateOptions struct {
	Title     string
	Body      string
	Draft     bool
	UseAI     bool
	Labels    []string
	Reviewers []string
}

func UpdatePR(ghc gh.Client, g git.Service, num int, opts PRUpdateOptions) error {
	// Get current PR details
	pr, err := ghc.GetPRDetails(num)
	if err != nil {
		return err
	}

	if opts.UseAI {
		// Get commit history since PR was created
		commits, err := g.Log(pr.Head.Ref, 0, true, false)
		if err != nil {
			return fmt.Errorf("failed to get commit history: %w", err)
		}

		// Get the diff for context
		diff, err := g.GetDiff()
		if err != nil {
			return fmt.Errorf("failed to get diff: %w", err)
		}

		// Initialize AI client
		client := ai.NewClient("")

		// Generate PR title if not explicitly provided
		if opts.Title == "" {
			title, err := client.GeneratePRTitle(commits, diff)
			if err != nil {
				return fmt.Errorf("failed to generate PR title: %w", err)
			}
			pr.Title = title
		}

		// Generate PR body based on commits and diff
		body, err := client.GeneratePRDescription(commits, diff)
		if err != nil {
			return fmt.Errorf("failed to generate PR body: %w", err)
		}
		pr.Body = body

		// Generate labels based on changes
		labels, err := client.GeneratePRLabels(commits, diff)
		if err != nil {
			return fmt.Errorf("failed to generate labels: %w", err)
		}
		opts.Labels = labels
	}

	// Update fields if specified
	if opts.Title != "" {
		pr.Title = opts.Title
	}
	if opts.Body != "" {
		pr.Body = opts.Body
	}
	pr.Draft = opts.Draft

	// Update the PR
	if err := ghc.UpdatePR(num, pr); err != nil {
		return err
	}

	// Update labels if specified
	if len(opts.Labels) > 0 {
		if err := ghc.AddLabels(num, opts.Labels); err != nil {
			return err
		}
	}

	// Update reviewers if specified
	if len(opts.Reviewers) > 0 {
		if err := ghc.RequestReviewers(num, opts.Reviewers); err != nil {
			return err
		}
	}

	return nil
}
