package app

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/git"
)

func StartBranch(g git.Service, newBranch string, push bool) error {
	repo, err := g.IsRepo()
	if err != nil || !repo {
		return fmt.Errorf("not a git repo")
	}

	db, err := g.DefaultBranch()
	if err != nil {
		db = "main"
	}

	if err := g.FetchAll(); err != nil {
		return err
	}

	if err := g.Checkout(db); err != nil {
		return err
	}

	if err := g.Pull(); err != nil {
		return err
	}

	if err := g.CreateBranch(newBranch); err != nil {
		return err
	}

	if err := g.Checkout(newBranch); err != nil {
		return err
	}

	if push {
		return g.Push(newBranch, false)
	}
	return nil
}
