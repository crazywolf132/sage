package app

import (
	"fmt"
	"strings"

	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
)

type CommitOptions struct {
	Message         string
	UseConventional bool
	AllowEmpty      bool
	PushAfterCommit bool
}

type CommitResult struct {
	ActualMessage string
	Pushed        bool
}

func Commit(g git.Service, opts CommitOptions) (*CommitResult, error) {
	isRepo, err := g.IsRepo()
	if err != nil {
		return nil, err
	}
	if !isRepo {
		return nil, fmt.Errorf("no a git repo")
	}

	if opts.Message == "" {
		// prompt
		msg, scope, ctype, err := ui.AskCommitMessage(opts.UseConventional)
		if err != nil {
			return nil, err
		}
		if opts.UseConventional {
			if scope != "" {
				opts.Message = fmt.Sprintf("%s(%s): %s", ctype, scope, msg)
			} else {
				opts.Message = fmt.Sprintf("%s: %s", ctype, msg)
			}
		} else {
			opts.Message = msg
		}
	} else if opts.UseConventional && !strings.Contains(opts.Message, ":") {
		opts.Message = "chore: " + opts.Message
	}

	if !opts.AllowEmpty {
		clean, err := g.IsClean()
		if err != nil {
			return nil, err
		}
		if clean {
			return nil, fmt.Errorf("no changes to commit; use --empty to allow empty")
		}
		if err := g.StageAll(); err != nil {
			return nil, err
		}
	}

	if err := g.Commit(opts.Message, opts.AllowEmpty); err != nil {
		return nil, err
	}
	res := &CommitResult{ActualMessage: opts.Message}

	if opts.PushAfterCommit {
		branch, err := g.CurrentBranch()
		if err != nil {
			return nil, err
		}
		if err := g.Push(branch, false); err != nil {
			return nil, err
		}
		res.Pushed = true
	}

	return res, nil
}
