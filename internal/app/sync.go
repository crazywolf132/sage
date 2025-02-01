package app

import (
	"fmt"
	"strings"

	"github.com/crazywolf132/sage/internal/git"
)

func SyncBranch(g git.Service, abort, cont bool) error {
	repo, err := g.IsRepo()
	if err != nil || !repo {
		return fmt.Errorf("not a git repo")
	}

	merging, err := g.IsMerging()
	if err != nil {
		return err
	}

	rebase, err := g.IsRebasing()
	if err != nil {
		return err
	}

	if abort {
		if merging {
			return g.MergeAbort()
		}
		if rebase {
			return g.RebaseAbort()
		}
		return fmt.Errorf("no merge or rebase to abort")
	}
	if cont {
		// not fully implemented
		return fmt.Errorf("merges or rebase continue not yet implemented here")
	}

	db, err := g.DefaultBranch()
	if err != nil {
		db = "main"
	}
	cur, err := g.CurrentBranch()
	if err != nil {
		return err
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
	if err := g.Checkout(cur); err != nil {
		return err
	}
	// do a merge
	if err := g.Merge(db); err != nil {
		if strings.Contains(err.Error(), "CONFLICT") {
			return fmt.Errorf("conflict - fix & run 'sage sync -c' or 'sage sync -a'")
		}
		return err
	}
	return nil
}
