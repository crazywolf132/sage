package app

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/gh"
	"github.com/crazywolf132/sage/internal/git"
)

// Create

// CreatePROpts with new fields
type CreatePROpts struct {
	Title       string
	Body        string
	Base        string
	Draft       bool
	Reviewers   []string
	Labels      []string
	UseTemplate bool
}

// CreatePullRequest orchestrates the entire creation process
func CreatePullRequest(g git.Service, ghc gh.Client, opts CreatePROpts) (*gh.PullRequest, error) {
	repo, err := g.IsRepo()
	if err != nil || !repo {
		return nil, fmt.Errorf("not a git repository")
	}
	curBranch, err := g.CurrentBranch()
	if err != nil {
		return nil, err
	}
	// push local changes first
	if err := g.Push(curBranch, false); err != nil {
		return nil, err
	}
	if opts.Base == "" {
		def, err := g.DefaultBranch()
		if err != nil {
			def = "main"
		}
		opts.Base = def
	}

	// If user wants to use GH PR template
	if opts.UseTemplate && opts.Body == "" {
		tmpl, _ := ghc.GetPRTemplate() // we must add a method for that
		if tmpl != "" {
			opts.Body = tmpl
		}
	}

	// create the PR
	pr, err := ghc.CreatePR(opts.Title, opts.Body, curBranch, opts.Base, opts.Draft)
	if err != nil {
		return nil, err
	}

	// If we want to set labels and reviewers, some of these might require separate API calls:
	if len(opts.Labels) > 0 {
		if e := ghc.AddLabels(pr.Number, opts.Labels); e != nil {
			// not fatal
		}
	}
	if len(opts.Reviewers) > 0 {
		if e := ghc.RequestReviewers(pr.Number, opts.Reviewers); e != nil {
			// not fatal
		}
	}

	return pr, nil
}

// List
func ListPRs(ghc gh.Client, state string) ([]gh.PullRequest, error) {
	return ghc.ListPRs(state)
}

// Merge
func MergePR(ghc gh.Client, prNum int, method string) error {
	return ghc.MergePR(prNum, method)
}

// Close
func ClosePR(ghc gh.Client, prNum int) error {
	return ghc.ClosePR(prNum)
}

// Checkout
func CheckoutPR(g git.Service, ghc gh.Client, prNum int) (string, error) {
	return ghc.CheckoutPR(prNum)
}

// Status
func GetPRDetails(ghc gh.Client, prNum int) (*gh.PullRequest, error) {
	return ghc.GetPRDetails(prNum)
}

// Todos
func ListUnresolvedThreads(ghc gh.Client, prNum int) ([]gh.UnresolvedThread, error) {
	return ghc.ListPRUnresolvedThreads(prNum)
}
