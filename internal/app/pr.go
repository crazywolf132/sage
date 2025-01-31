package app

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/gh"
	"github.com/crazywolf132/sage/internal/git"
)

// Create
type CreatePROpts struct {
	Title string
	Body  string
	Base  string
	Draft bool
}

func CreatePullRequest(g git.Service, ghc gh.Client, opts CreatePROpts) (*gh.PullRequest, error) {
	repo, err := g.IsRepo()
	if err != nil || !repo {
		return nil, fmt.Errorf("not a repo or error checking repo: %v", err)
	}
	branch, err := g.CurrentBranch()
	if err != nil {
		return nil, err
	}
	if err := g.Push(branch, false); err != nil {
		return nil, err
	}
	if opts.Base == "" {
		db, err2 := g.DefaultBranch()
		if err2 != nil {
			db = "main"
		}
		opts.Base = db
	}
	return ghc.CreatePR(opts.Title, opts.Body, branch, opts.Base, opts.Draft)
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
