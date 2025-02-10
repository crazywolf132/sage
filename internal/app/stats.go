package app

import (
	"fmt"
	"sort"
	"time"

	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
)

type StatsOptions struct {
	TimeRange string
	Limit     int
	Detailed  bool
}

type FileStats struct {
	Path         string
	Changes      int
	LastModified time.Time
	Authors      map[string]int
}

type AuthorStats struct {
	Name         string
	Commits      int
	Additions    int
	Deletions    int
	FilesChanged map[string]int
}

type BranchStats struct {
	Name           string
	LastCommit     time.Time
	CommitCount    int
	MergeConflicts int
}

func GetStats(g git.Service, opts StatsOptions) error {
	repo, err := g.IsRepo()
	if err != nil || !repo {
		return fmt.Errorf("not a git repository")
	}

	// Get the time range for filtering
	var since time.Time
	switch opts.TimeRange {
	case "day":
		since = time.Now().AddDate(0, 0, -1)
	case "week":
		since = time.Now().AddDate(0, 0, -7)
	case "month":
		since = time.Now().AddDate(0, -1, 0)
	case "year":
		since = time.Now().AddDate(-1, 0, 0)
	}

	// Initialize statistics maps
	fileStats := make(map[string]*FileStats)
	authorStats := make(map[string]*AuthorStats)
	branchStats := make(map[string]*BranchStats)

	// Get branch information
	branches, err := g.ListBranches()
	if err != nil {
		return fmt.Errorf("failed to list branches: %w", err)
	}

	// Collect branch statistics
	for _, branch := range branches {
		lastCommit, err := g.GetBranchLastCommit(branch)
		if err != nil {
			continue
		}

		commitCount, err := g.GetBranchCommitCount(branch)
		if err != nil {
			continue
		}

		conflicts, err := g.GetBranchMergeConflicts(branch)
		if err != nil {
			conflicts = 0
		}

		branchStats[branch] = &BranchStats{
			Name:           branch,
			LastCommit:     lastCommit,
			CommitCount:    commitCount,
			MergeConflicts: conflicts,
		}
	}

	// Get all commits with stats
	log, err := g.Log("", 0, true, true)
	if err != nil {
		return fmt.Errorf("failed to get git log: %w", err)
	}

	// Parse the log and collect statistics
	commits := parseGitLog(log, true)

	// Process each commit
	for _, commit := range commits {
		// Skip if outside time range
		if !since.IsZero() && commit.Date.Before(since) {
			continue
		}

		// Update author stats
		author, ok := authorStats[commit.AuthorName]
		if !ok {
			author = &AuthorStats{
				Name:         commit.AuthorName,
				FilesChanged: make(map[string]int),
			}
			authorStats[commit.AuthorName] = author
		}
		author.Commits++
		author.Additions += commit.Stats.Added
		author.Deletions += commit.Stats.Deleted

		// Update file stats
		for file, changes := range commit.Stats.Files {
			fs, ok := fileStats[file]
			if !ok {
				fs = &FileStats{
					Path:    file,
					Authors: make(map[string]int),
				}
				fileStats[file] = fs
			}
			fs.Changes += changes
			fs.LastModified = commit.Date
			fs.Authors[commit.AuthorName]++
			author.FilesChanged[file]++
		}
	}

	// Print repository overview
	fmt.Printf("\n%s Repository Statistics\n\n", ui.Sage("📊"))

	// Print file statistics
	printFileStats(fileStats, opts.Limit)

	// Print author statistics
	printAuthorStats(authorStats, opts.Limit)

	if opts.Detailed {
		// Print branch statistics
		printBranchStats(branchStats, opts.Limit)

		// Print additional metrics
		printDetailedMetrics(fileStats, authorStats, branchStats)
	}

	return nil
}

func printFileStats(stats map[string]*FileStats, limit int) {
	fmt.Printf("%s Most Active Files:\n\n", ui.Sage("📁"))

	// Convert map to slice for sorting
	files := make([]*FileStats, 0, len(stats))
	for _, fs := range stats {
		files = append(files, fs)
	}

	// Sort by number of changes
	sort.Slice(files, func(i, j int) bool {
		return files[i].Changes > files[j].Changes
	})

	// Print top N files
	for i, fs := range files {
		if i >= limit {
			break
		}
		fmt.Printf("  %s %s\n", ui.Yellow("•"), fs.Path)
		fmt.Printf("    Changes: %d, Last Modified: %s\n", fs.Changes, fs.LastModified.Format("2006-01-02"))
		fmt.Printf("    Contributors: %d\n", len(fs.Authors))
	}
	fmt.Println()
}

func printAuthorStats(stats map[string]*AuthorStats, limit int) {
	fmt.Printf("%s Top Contributors:\n\n", ui.Sage("👥"))

	// Convert map to slice for sorting
	authors := make([]*AuthorStats, 0, len(stats))
	for _, as := range stats {
		authors = append(authors, as)
	}

	// Sort by number of commits
	sort.Slice(authors, func(i, j int) bool {
		return authors[i].Commits > authors[j].Commits
	})

	// Print top N authors
	for i, as := range authors {
		if i >= limit {
			break
		}
		fmt.Printf("  %s %s\n", ui.Yellow("•"), as.Name)
		fmt.Printf("    Commits: %d, Files Changed: %d\n", as.Commits, len(as.FilesChanged))
		fmt.Printf("    Added: %d, Deleted: %d\n", as.Additions, as.Deletions)
	}
	fmt.Println()
}

func printBranchStats(stats map[string]*BranchStats, limit int) {
	fmt.Printf("%s Branch Activity:\n\n", ui.Sage("🌿"))

	// Convert map to slice for sorting
	branches := make([]*BranchStats, 0, len(stats))
	for _, bs := range stats {
		branches = append(branches, bs)
	}

	// Sort by commit count
	sort.Slice(branches, func(i, j int) bool {
		return branches[i].CommitCount > branches[j].CommitCount
	})

	// Print top N branches
	for i, bs := range branches {
		if i >= limit {
			break
		}
		fmt.Printf("  %s %s\n", ui.Yellow("•"), bs.Name)
		fmt.Printf("    Commits: %d, Last Activity: %s\n", bs.CommitCount, bs.LastCommit.Format("2006-01-02"))
		if bs.MergeConflicts > 0 {
			fmt.Printf("    Merge Conflicts: %d\n", bs.MergeConflicts)
		}
	}
	fmt.Println()
}

func printDetailedMetrics(fileStats map[string]*FileStats, authorStats map[string]*AuthorStats, branchStats map[string]*BranchStats) {
	fmt.Printf("%s Additional Metrics:\n\n", ui.Sage("📈"))

	// Calculate average changes per file
	totalChanges := 0
	for _, fs := range fileStats {
		totalChanges += fs.Changes
	}
	avgChanges := float64(totalChanges) / float64(len(fileStats))

	// Calculate average commits per author
	totalCommits := 0
	for _, as := range authorStats {
		totalCommits += as.Commits
	}
	avgCommits := float64(totalCommits) / float64(len(authorStats))

	fmt.Printf("  %s Repository Health:\n", ui.Yellow("•"))
	fmt.Printf("    Total Files: %d\n", len(fileStats))
	fmt.Printf("    Total Contributors: %d\n", len(authorStats))
	fmt.Printf("    Average Changes per File: %.2f\n", avgChanges)
	fmt.Printf("    Average Commits per Author: %.2f\n", avgCommits)
	fmt.Println()
}
