package app

import (
	"fmt"
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
		lines := strings.Split(strings.TrimRight(porcelain, "\n"), "\n")
		for _, ln := range lines {
			if len(ln) < 4 { // Need at least "? filename" or "XY filename"
				continue
			}

			indexStatus := ln[0]    // First character is always index status
			workTreeStatus := ln[1] // Second character is always worktree status

			pathStart := 2
			for pathStart < len(ln) && ln[pathStart] == ' ' {
				pathStart++
			}

			if pathStart >= len(ln) {
				continue
			}

			path := ln[pathStart:]

			if strings.Contains(path, " -> ") {
				parts := strings.Split(path, " -> ")
				if len(parts) == 2 {
					path = parts[1]
				}
			}

			symbol, desc := interpretStatus(indexStatus, workTreeStatus)
			changes = append(changes, FileChange{
				Symbol:      symbol,
				File:        path,
				Description: desc,
			})
		}
	}
	return &RepoStatus{Branch: br, Changes: changes}, nil
}

// interpretStatus interprets the status codes from git status --porcelain=v1
// indexStatus is the status in the staging area (first character)
// workTreeStatus is the status in the working tree (second character)
func interpretStatus(indexStatus, workTreeStatus byte) (string, string) {
	if indexStatus == '?' && workTreeStatus == '?' {
		return "?", "Untracked"
	}

	switch indexStatus {
	case 'M':
		if workTreeStatus == 'M' {
			return "M", "Staged+Unstaged Modified"
		}
		return "M", "Staged Modified"
	case 'A':
		if workTreeStatus == 'M' {
			return "M", "Staged Added, with modifications"
		}
		if workTreeStatus == 'D' {
			return "D", "Staged Added, but deleted"
		}
		return "A", "Staged Added"
	case 'D':
		if workTreeStatus == 'M' {
			return "M", "Staged Deleted, but modified"
		}
		return "D", "Staged Deleted"
	case 'R':
		if workTreeStatus == 'M' {
			return "R", "Staged Renamed, with modifications"
		}
		return "R", "Staged Renamed"
	case 'C':
		if workTreeStatus == 'M' {
			return "C", "Staged Copied, with modifications"
		}
		return "C", "Staged Copied"
	case ' ':
		switch workTreeStatus {
		case 'M':
			return "M", "Unstaged Modified"
		case 'D':
			return "D", "Unstaged Deleted"
		case 'A':
			return "A", "Unstaged Added"
		}
	}

	return " ", fmt.Sprintf("Unknown Status: index=[%c] worktree=[%c]", indexStatus, workTreeStatus)
}

// Helper function to get status description
func getStatusDescription(status byte) string {
	switch status {
	case 'M':
		return "Modified"
	case 'A':
		return "Added"
	case 'D':
		return "Deleted"
	case 'R':
		return "Renamed"
	default:
		return "Unknown"
	}
}
