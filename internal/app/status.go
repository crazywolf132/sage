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

			// Handle renamed files
			if strings.Contains(path, " -> ") {
				parts := strings.Split(path, " -> ")
				path = parts[1] // Use the new path
			}

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
	// First character represents staging area
	// Second character represents working tree
	switch code {
	case "M ":
		return "M", "Staged Modified"
	case " M":
		return "M", "Modified"
	case "A ":
		return "A", "Staged Added"
	case "AM":
		return "M", "Staged Added, with modifications"
	case "D ":
		return "D", "Staged Deleted"
	case " D":
		return "D", "Deleted"
	case "R ":
		return "R", "Staged Renamed"
	case "??":
		return "?", "Untracked"
	default:
		return " ", "Unknown"
	}
}
