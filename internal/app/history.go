package app

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/crazywolf132/sage/internal/git"
)

type HistoryOptions struct {
	Limit     int
	ShowStats bool
	ShowAll   bool
	Branch    string
}

type CommitStats struct {
	Added    int
	Deleted  int
	Modified int
	Files    map[string]int // Maps file paths to number of changes
}

type CommitInfo struct {
	Hash       string
	ShortHash  string
	AuthorName string
	Date       time.Time
	Message    string
	Stats      CommitStats
}

type HistoryResult struct {
	BranchName string
	Commits    []CommitInfo
}

func GetHistory(g git.Service, branch string, limit int, showStats, showAll bool) (*HistoryResult, error) {
	repo, err := g.IsRepo()
	if err != nil || !repo {
		return nil, fmt.Errorf("not a git repository")
	}

	if branch == "" {
		branch, err = g.CurrentBranch()
		if err != nil {
			return nil, err
		}
	}
	log, err := g.Log(branch, limit, showStats, showAll)
	if err != nil {
		return nil, err
	}
	commits := parseGitLog(log, showStats)
	return &HistoryResult{
		BranchName: branch,
		Commits:    commits,
	}, nil
}

func parseGitLog(log string, stats bool) []CommitInfo {
	var commits []CommitInfo
	var current *CommitInfo

	lines := strings.Split(strings.TrimSpace(log), "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.Contains(line, "\x00") {
			// This is a commit line
			if current != nil {
				commits = append(commits, *current)
			}

			parts := strings.Split(line, "\x00")
			if len(parts) < 4 {
				continue
			}

			// Parse timestamp
			timestamp, _ := strconv.ParseInt(parts[2], 10, 64)
			date := time.Unix(timestamp, 0)

			current = &CommitInfo{
				Hash:       parts[0],
				ShortHash:  parts[0][:7],
				AuthorName: parts[1],
				Date:       date,
				Message:    parts[3],
				Stats: CommitStats{
					Files: make(map[string]int),
				},
			}

			// Look ahead for stats if requested
			if stats && i+3 < len(lines) {
				// Skip blank line
				i++
				// Next lines should be stats until we hit a blank line or another commit
				for i+1 < len(lines) {
					statLine := lines[i+1]
					if statLine == "" || strings.Contains(statLine, "\x00") {
						break
					}
					parts := strings.Fields(statLine)
					if len(parts) >= 3 {
						added, _ := strconv.Atoi(parts[0])
						deleted, _ := strconv.Atoi(parts[1])
						file := parts[2]
						current.Stats.Added += added
						current.Stats.Deleted += deleted
						current.Stats.Modified++
						current.Stats.Files[file] = added + deleted
					}
					i++
				}
			}
		}
	}

	// Don't forget the last commit
	if current != nil {
		commits = append(commits, *current)
	}

	return commits
}
