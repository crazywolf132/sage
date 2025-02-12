package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ShellGit implements the Service interface using shell commands to interact with Git
type ShellGit struct{}

// NewShellGit creates a new instance of ShellGit that implements the Service interface
func NewShellGit() Service {
	return &ShellGit{}
}

// validateRef validates a Git reference name
func validateRef(ref string) error {
	if ref == "" {
		return fmt.Errorf("empty reference name")
	}
	// Check for common Git command injection patterns
	if strings.Contains(ref, "&&") ||
		strings.Contains(ref, "||") ||
		strings.Contains(ref, ";") ||
		strings.Contains(ref, "|") ||
		strings.Contains(ref, ">") ||
		strings.Contains(ref, "<") ||
		strings.Contains(ref, "`") ||
		strings.Contains(ref, "$") ||
		strings.Contains(ref, "(") ||
		strings.Contains(ref, ")") {
		return fmt.Errorf("invalid characters in reference name")
	}
	return nil
}

// validatePath validates a file path
func validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}
	// Check for path traversal and command injection
	if strings.Contains(path, "..") ||
		strings.Contains(path, "&&") ||
		strings.Contains(path, "||") ||
		strings.Contains(path, ";") ||
		strings.Contains(path, "|") ||
		strings.Contains(path, ">") ||
		strings.Contains(path, "<") ||
		strings.Contains(path, "`") ||
		strings.Contains(path, "$") ||
		strings.Contains(path, "(") ||
		strings.Contains(path, ")") {
		return fmt.Errorf("invalid characters in path")
	}
	return nil
}

// Run executes a git command with the given arguments and returns its output
func (s *ShellGit) Run(args ...string) (string, error) {
	return s.run(args...)
}

// run is the internal implementation of Run
func (s *ShellGit) run(args ...string) (string, error) {
	// Validate all arguments
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue // Skip flags
		}
		if err := validateRef(arg); err != nil {
			return "", fmt.Errorf("invalid argument: %w", err)
		}
	}

	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, stderr.String())
	}
	return string(out), nil
}

// runInteractive executes a git command in interactive mode, connecting it to the terminal's
// standard input, output, and error streams
func (s *ShellGit) runInteractive(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// IsRepo checks if the current directory is a git repository
// Returns true if it is, false if not, and any error encountered
func (s *ShellGit) IsRepo() (bool, error) {
	_, err := s.run("rev-parse", "--git-dir")
	if err != nil && strings.Contains(err.Error(), "not a git repository") {
		return false, nil
	}
	return err == nil, nil
}

// CurrentBranch returns the name of the current git branch
func (s *ShellGit) CurrentBranch() (string, error) {
	out, err := s.run("rev-parse", "--abbrev-ref", "HEAD")
	return strings.TrimSpace(out), err
}

// IsClean checks if the working directory is clean (no uncommitted changes)
// Returns true if clean, false if there are changes
func (s *ShellGit) IsClean() (bool, error) {
	out, err := s.run("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "", nil
}

// StageAll stages all changes in the working directory
func (s *ShellGit) StageAll() error {
	_, err := s.run("add", ".")
	return err
}

// Commit creates a new commit with the given message
// If allowEmpty is true, allows creating empty commits
// If stageAll is true, automatically stages all changes before committing
func (s *ShellGit) Commit(msg string, allowEmpty bool, stageAll bool) error {
	args := []string{"commit"}
	if stageAll {
		args = append(args, "-a")
	}
	args = append(args, "-m", msg)
	if allowEmpty {
		args = append(args, "--allow-empty")
	}
	_, err := s.run(args...)
	return err
}

// Push pushes the specified branch to the remote repository
// If force is true, performs a force push
func (s *ShellGit) Push(branch string, force bool) error {
	if err := validateRef(branch); err != nil {
		return fmt.Errorf("invalid branch name: %w", err)
	}

	args := []string{"push", "origin", branch}
	if force {
		args = []string{"push", "--force", "origin", branch}
	}
	_, err := s.run(args...)
	return err
}

// PushWithLease pushes the specified branch to the remote repository using --force-with-lease
// This is safer than force push as it ensures we don't overwrite changes we haven't seen
func (s *ShellGit) PushWithLease(branch string) error {
	if err := validateRef(branch); err != nil {
		return fmt.Errorf("invalid branch name: %w", err)
	}

	_, err := s.run("push", "--force-with-lease", "origin", branch)
	return err
}

// DefaultBranch returns the name of the default branch (usually main or master)
func (s *ShellGit) DefaultBranch() (string, error) {
	out, err := s.run("symbolic-ref", "refs/remotes/origin/HEAD")
	if err != nil {
		return "", err
	}
	out = strings.TrimSpace(out)
	parts := strings.Split(out, "/")
	if len(parts) < 1 {
		return "main", nil
	}
	return parts[len(parts)-1], nil
}

// MergedBranches returns a list of branches that have been merged into the specified base branch
func (s *ShellGit) MergedBranches(base string) ([]string, error) {
	out, err := s.run("branch", "--merged", base)
	if err != nil {
		return nil, err
	}
	var res []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "* "))
		if line != "" {
			res = append(res, line)
		}
	}
	return res, nil
}

// DeleteBranch deletes the specified branch
// If the branch is not fully merged, attempts a force delete
func (s *ShellGit) DeleteBranch(name string) error {
	_, err := s.run("branch", "-d", name)
	if err != nil && strings.Contains(err.Error(), "is not fully merged") {
		_, err2 := s.run("branch", "-D", name)
		if err2 == nil {
			return nil
		}
		return err
	}
	return err
}

// FetchAll fetches all remotes and prunes deleted remote branches
func (s *ShellGit) FetchAll() error {
	_, err := s.run("fetch", "--all", "--prune")
	return err
}

// Checkout switches to the specified branch or commit
func (s *ShellGit) Checkout(name string) error {
	if err := validateRef(name); err != nil {
		return fmt.Errorf("invalid checkout target: %w", err)
	}
	_, err := s.run("checkout", name)
	return err
}

// Pull performs a git pull in interactive mode
func (s *ShellGit) Pull() error {
	return s.runInteractive("pull")
}

// PullFF performs a fast-forward only pull
func (s *ShellGit) PullFF() error {
	return s.runInteractive("pull", "--ff-only")
}

// PullRebase performs a pull with rebase
func (s *ShellGit) PullRebase() error {
	return s.runInteractive("pull", "--rebase")
}

// CreateBranch creates a new branch with the specified name
func (s *ShellGit) CreateBranch(name string) error {
	if err := validateRef(name); err != nil {
		return fmt.Errorf("invalid branch name: %w", err)
	}
	_, err := s.run("branch", name)
	return err
}

// Merge merges the specified base branch into the current branch
func (s *ShellGit) Merge(base string) error {
	return s.runInteractive("merge", base)
}

// MergeAbort aborts an in-progress merge
func (s *ShellGit) MergeAbort() error {
	return s.runInteractive("merge", "--abort")
}

// IsMerging checks if a merge is currently in progress
func (s *ShellGit) IsMerging() (bool, error) {
	_, err := s.run("rev-parse", "--verify", "MERGE_HEAD")
	if err != nil && strings.Contains(err.Error(), "not a valid object name") {
		return false, nil
	}
	return (err == nil), nil
}

// RebaseAbort aborts an in-progress rebase
func (s *ShellGit) RebaseAbort() error {
	return s.runInteractive("rebase", "--abort")
}

// IsRebasing checks if a rebase is currently in progress
func (s *ShellGit) IsRebasing() (bool, error) {
	_, err := s.run("rev-parse", "--verify", "REBASE_HEAD")
	if err != nil && strings.Contains(err.Error(), "not a valid object name") {
		return false, nil
	}
	return (err == nil), nil
}

// StatusPorcelain returns the git status in porcelain format
func (s *ShellGit) StatusPorcelain() (string, error) {
	return s.run("status", "--porcelain")
}

// ResetSoft performs a soft reset to the specified reference
func (s *ShellGit) ResetSoft(ref string) error {
	_, err := s.run("reset", "--soft", ref)
	return err
}

// ListBranches returns a list of all local branches
func (s *ShellGit) ListBranches() ([]string, error) {
	out, err := s.run("branch", "--sort=-committerdate", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	return lines, nil
}

// Log returns the git log with custom formatting
// If branch is specified, shows log for that branch
// If limit > 0, limits the number of entries (unless all is true)
// If stats is true, includes numstat information
func (s *ShellGit) Log(branch string, limit int, stats, all bool) (string, error) {
	// Build the git log command with a custom format
	args := []string{
		"log",
		"--pretty=format:%H%x00%an%x00%at%x00%s", // Use null bytes as separators
	}

	if limit > 0 && !all {
		args = append(args, "-n", strconv.Itoa(limit))
	}

	if stats {
		args = append(args, "--numstat")
	}

	if branch != "" {
		args = append(args, branch)
	}

	out, err := s.run(args...)
	if err != nil {
		return "", err
	}

	return out, nil
}

// GetDiff returns the current diff
// First checks for staged changes, then unstaged if no staged changes exist
func (s *ShellGit) GetDiff() (string, error) {
	// First check if there are staged changes
	stagedDiff, err := s.run("diff", "--cached")
	if err != nil {
		return "", fmt.Errorf("failed to get staged changes: %w", err)
	}

	// If there are staged changes, return those
	if strings.TrimSpace(stagedDiff) != "" {
		return stagedDiff, nil
	}

	// If no staged changes, get unstaged changes
	unstagedDiff, err := s.run("diff")
	if err != nil {
		return "", fmt.Errorf("failed to get unstaged changes: %w", err)
	}

	return unstagedDiff, nil
}

// SquashCommits performs an interactive rebase to squash commits from the specified start commit
func (s *ShellGit) SquashCommits(startCommit string) error {
	return s.runInteractive("rebase", "-i", startCommit)
}

// IsHeadBranch checks if the specified branch is the default branch
func (s *ShellGit) IsHeadBranch(branch string) (bool, error) {
	defaultBranch, err := s.DefaultBranch()
	if err != nil {
		return false, err
	}
	return branch == defaultBranch, nil
}

// GetFirstCommit returns the hash of the first commit in the repository
func (s *ShellGit) GetFirstCommit() (string, error) {
	out, err := s.run("rev-list", "--max-parents=0", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get first commit: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// RunInteractive runs a git command in interactive mode with the specified arguments
func (g *ShellGit) RunInteractive(cmd string, args ...string) error {
	cmdArgs := append([]string{cmd}, args...)
	return g.runInteractive(cmdArgs...)
}

// IsPathStaged checks if the specified path is staged in git
func (s *ShellGit) IsPathStaged(path string) (bool, error) {
	// First check if the path exists in the working tree
	out, err := s.run("ls-files", path)
	if err != nil {
		// If path doesn't exist, it's not staged
		return false, nil
	}

	// If path exists, check if it's staged
	out, err = s.run("diff", "--cached", "--name-only", "--", path)
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(out) != "", nil
}

// StageAllExcept stages all changes except those in the specified paths
func (s *ShellGit) StageAllExcept(excludePaths []string) error {
	// First, get all changes
	out, err := s.run("status", "--porcelain")
	if err != nil {
		return err
	}

	// Process each changed file
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}

		// Status format is XY PATH or XY PATH -> PATH2 for renames
		// X is status in staging area, Y is status in working tree
		status := line[:2]
		path := strings.TrimSpace(line[3:])

		// Handle renamed files
		if strings.Contains(path, " -> ") {
			parts := strings.Split(path, " -> ")
			path = parts[1] // Use the new path
		}

		// Check if this path should be excluded
		shouldExclude := false
		for _, excludePath := range excludePaths {
			if strings.HasPrefix(path, excludePath) {
				shouldExclude = true
				break
			}
		}

		if !shouldExclude {
			// Only add if the file is modified, added, or deleted in working tree
			// Skip if it's already staged (X is not space)
			if status[0] == ' ' && (status[1] == 'M' || status[1] == 'A' || status[1] == 'D') {
				_, err := s.run("add", "--", path)
				if err != nil {
					return fmt.Errorf("failed to stage %s: %w", path, err)
				}
			}
		}
	}
	return nil
}

// GetBranchLastCommit returns the timestamp of the last commit on the specified branch
func (s *ShellGit) GetBranchLastCommit(branch string) (time.Time, error) {
	out, err := s.run("log", "-1", "--format=%at", branch)
	if err != nil {
		return time.Time{}, err
	}
	timestamp, err := strconv.ParseInt(strings.TrimSpace(out), 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(timestamp, 0), nil
}

// GetBranchCommitCount returns the total number of commits in the specified branch
func (s *ShellGit) GetBranchCommitCount(branch string) (int, error) {
	out, err := s.run("rev-list", "--count", branch)
	if err != nil {
		return 0, err
	}
	count, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetBranchMergeConflicts returns the number of potential merge conflicts
// between the specified branch and the default branch
func (s *ShellGit) GetBranchMergeConflicts(branch string) (int, error) {
	// Get merge base with default branch
	defaultBranch, err := s.DefaultBranch()
	if err != nil {
		return 0, err
	}

	base, err := s.run("merge-base", defaultBranch, branch)
	if err != nil {
		return 0, err
	}

	// Try a merge and count conflicts
	out, err := s.run("merge-tree", strings.TrimSpace(base), defaultBranch, branch)
	if err != nil {
		return 0, err
	}

	// Count conflict markers
	return strings.Count(out, "<<<<<<<"), nil
}

// MergeContinue continues a merge operation after conflicts have been resolved
// Returns an error if there are still unresolved conflicts
func (s *ShellGit) MergeContinue() (string, error) {
	out, err := s.run("merge", "--continue")
	if err != nil {
		// if error message indicates conflicts, inform the user.
		if strings.Contains(err.Error(), "conflict") {
			return "", fmt.Errorf("merge conflicts encountered; please resolve conflicts and run 'sage sync --continue'")
		}
		return "", fmt.Errorf("failed to continue merge: %w", err)
	}
	return out, nil
}

// RebaseContinue continues a rebase operation after conflicts have been resolved
// Returns an error if there are still unresolved conflicts
func (s *ShellGit) RebaseContinue() (string, error) {
	out, err := s.run("rebase", "--continue")
	if err != nil {
		if strings.Contains(err.Error(), "confict") {
			return "", fmt.Errorf("rebase conflicts encountered; please resolve conflicts and run 'sage sync --continue'")
		}
		return "", fmt.Errorf("failed to continue rebase: %w", err)
	}
	return out, nil
}

// ListConflictedFiles returns a string listing files with unresolved conflicts.
// It runs: git diff --name-only --diff-filter=U
func (s *ShellGit) ListConflictedFiles() (string, error) {
	out, err := s.run("diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Stash saves the current changes to the stash with a message
func (g *ShellGit) Stash(message string) error {
	_, err := g.run("stash", "push", "-m", message)
	return err
}

// StashPop applies and removes the most recent stash
func (g *ShellGit) StashPop() error {
	_, err := g.run("stash", "pop")
	return err
}

// StashList returns a list of stashes
func (g *ShellGit) StashList() ([]string, error) {
	out, err := g.run("stash", "list")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return []string{}, nil
	}
	return strings.Split(strings.TrimSpace(out), "\n"), nil
}

// GetMergeBase finds the best common ancestor between two branches
func (g *ShellGit) GetMergeBase(branch1, branch2 string) (string, error) {
	out, err := g.run("merge-base", branch1, branch2)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// GetCommitCount returns the number of commits in the given revision range
func (g *ShellGit) GetCommitCount(revisionRange string) (int, error) {
	out, err := g.run("rev-list", "--count", revisionRange)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(out))
}

// GetBranchDivergence returns the total number of commits that differ between two branches
func (g *ShellGit) GetBranchDivergence(branch1, branch2 string) (int, error) {
	// Get the merge base
	base, err := g.GetMergeBase(branch1, branch2)
	if err != nil {
		return 0, err
	}

	// Count commits in each branch since the merge base
	count1, err := g.GetCommitCount(fmt.Sprintf("%s..%s", base, branch1))
	if err != nil {
		return 0, err
	}

	count2, err := g.GetCommitCount(fmt.Sprintf("%s..%s", base, branch2))
	if err != nil {
		return 0, err
	}

	return count1 + count2, nil
}

// GetCommitHash returns the commit hash for the given reference
func (g *ShellGit) GetCommitHash(ref string) (string, error) {
	out, err := g.run("rev-parse", ref)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// IsAncestor checks if commit1 is an ancestor of commit2
func (g *ShellGit) IsAncestor(commit1, commit2 string) (bool, error) {
	_, err := g.run("merge-base", "--is-ancestor", commit1, commit2)
	if err != nil {
		if strings.Contains(err.Error(), "exit status 1") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
