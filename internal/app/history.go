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
	lines := strings.Split(strings.TrimSpace(log), "\n")

	var commits []CommitInfo
	var current *CommitInfo
	var collectingStats bool

	for _, ln := range lines {
		if strings.Contains(ln, "\x00") {
			// new commit line
			if current != nil {
				commits = append(commits, *current)
			}
			parts := strings.Split(ln, "\x00")
			// parts: 0=sha, 1=author, 2=timestamp, 3=message
			if len(parts) < 4 {
				continue
			}
			sha := parts[0]
			author := parts[1]
			tUnix, _ := strconv.ParseInt(parts[2], 10, 64)
			msg := parts[3]

			current = &CommitInfo{
				Hash:       sha,
				ShortHash:  sha[:7],
				AuthorName: author,
				Date:       time.Unix(tUnix, 0),
				Message:    msg,
			}
			collectingStats = stats
		} else if collectingStats && ln != "" {
			// expecting lines like "added deleted file"
			fs := strings.Fields(ln)
			if len(fs) >= 3 {
				added, _ := strconv.Atoi(fs[0])
				deleted, _ := strconv.Atoi(fs[1])
				if added > 0 {
					current.Stats.Added += added
				}
				if deleted > 0 {
					current.Stats.Deleted += deleted
				}
				if added > 0 && deleted > 0 {
					current.Stats.Modified++
				}
			}
		}
	}
	if current != nil {
		commits = append(commits, *current)
	}
	return commits
}
