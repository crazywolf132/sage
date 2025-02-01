package app

import (
	"strings"

	"github.com/crazywolf132/sage/internal/git"
)

type FileChange struct {
	Symbol      string
	File        string
	Description string
}

type RepoStatus struct {
	Branch  string
	Changes []FileChange
}

func GetRepoStatus(g git.Service) (*RepoStatus, error) {
	repo, err := g.IsRepo()
	if err != nil || !repo {
		return nil, err
	}

	br, err := g.CurrentBranch()
	if err != nil {
		return nil, err
	}

	porcelain, err := g.StatusPorcelain()
	if err != nil {
		return nil, err
	}

	var changes []FileChange
	if porcelain != "" {
		lines := strings.Split(strings.TrimSpace(porcelain), "\n")
		for _, ln := range lines {
			if len(ln) < 4 {
				continue
			}
			code := ln[:2]
			path := strings.TrimSpace(ln[3:])
			symbol, desc := interpretStatus(code)
			changes = append(changes, FileChange{
				Symbol:      symbol,
				File:        path,
				Description: desc,
			})
		}
	}
	return &RepoStatus{Branch: br, Changes: changes}, nil
}

func interpretStatus(code string) (string, string) {
	switch code {
	case "M ", " M":
		return "M", "Modified"
	case "A ", "AM":
		return "A", "Added"
	case "D ", " D":
		return "D", "Deleted"
	case "R ":
		return "R", "Renamed"
	case "??":
		return "?", "Untracked"
	default:
		return " ", "Unknown"
	}
}
