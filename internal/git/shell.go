package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type shellGit struct{}

func NewShellGit() Service {
	return &shellGit{}
}

func (s *shellGit) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, stderr.String())
	}
	return string(out), nil
}

func (s *shellGit) runInteractive(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (s *shellGit) IsRepo() (bool, error) {
	_, err := s.run("rev-parse", "--git-dir")
	if err != nil && strings.Contains(err.Error(), "not a git repository") {
		return false, nil
	}
	return err == nil, nil
}

func (s *shellGit) CurrentBranch() (string, error) {
	out, err := s.run("rev-parse", "--abbrev-ref", "HEAD")
	return strings.TrimSpace(out), err
}

func (s *shellGit) IsClean() (bool, error) {
	out, err := s.run("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "", nil
}

func (s *shellGit) StageAll() error {
	_, err := s.run("add", ".")
	return err
}

func (s *shellGit) Commit(msg string, allowEmpty bool) error {
	args := []string{"commit", "-m", msg}
	if allowEmpty {
		args = append(args, "--allow-empty")
	}
	_, err := s.run(args...)
	return err
}

func (s *shellGit) Push(branch string, force bool) error {
	args := []string{"push", "origin", branch}
	if force {
		args = []string{"push", "--force", "origin", branch}
	}
	_, err := s.run(args...)
	return err
}

func (s *shellGit) DefaultBranch() (string, error) {
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

func (s *shellGit) MergedBranches(base string) ([]string, error) {
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

func (s *shellGit) DeleteBranch(name string) error {
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

func (s *shellGit) FetchAll() error {
	_, err := s.run("fetch", "--all", "--prune")
	return err
}

func (s *shellGit) Checkout(name string) error {
	_, err := s.run("checkout", name)
	return err
}

func (s *shellGit) Pull() error {
	return s.runInteractive("pull")
}

func (s *shellGit) PullFF() error {
	return s.runInteractive("pull", "--ff-only")
}

func (s *shellGit) PullRebase() error {
	return s.runInteractive("pull", "--rebase")
}

func (s *shellGit) CreateBranch(name string) error {
	_, err := s.run("branch", name)
	return err
}

func (s *shellGit) Merge(base string) error {
	return s.runInteractive("merge", base)
}

func (s *shellGit) MergeAbort() error {
	return s.runInteractive("merge", "--abort")
}

func (s *shellGit) IsMerging() (bool, error) {
	_, err := s.run("rev-parse", "--verify", "MERGE_HEAD")
	if err != nil && strings.Contains(err.Error(), "not a valid object name") {
		return false, nil
	}
	return (err == nil), nil
}

func (s *shellGit) RebaseAbort() error {
	return s.runInteractive("rebase", "--abort")
}

func (s *shellGit) IsRebasing() (bool, error) {
	_, err := s.run("rev-parse", "--verify", "REBASE_HEAD")
	if err != nil && strings.Contains(err.Error(), "not a valid object name") {
		return false, nil
	}
	return (err == nil), nil
}

func (s *shellGit) StatusPorcelain() (string, error) {
	return s.run("status", "--porcelain")
}

func (s *shellGit) ResetSoft(ref string) error {
	_, err := s.run("reset", "--soft", ref)
	return err
}

func (s *shellGit) ListBranches() ([]string, error) {
	out, err := s.run("branch", "--list", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	return lines, nil
}

func (s *shellGit) Log(branch string, limit int, stats, all bool) (string, error) {
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

func (s *shellGit) GetDiff() (string, error) {
	output, err := s.run("diff", "--staged")
	if err != nil {
		// If there's an error, try getting unstaged changes
		output, err = s.run("diff")
		if err != nil {
			return "", err
		}
	}
	return output, nil
}

func (s *shellGit) SquashCommits(startCommit string) error {
	return s.runInteractive("rebase", "-i", startCommit)
}

func (s *shellGit) IsHeadBranch(branch string) (bool, error) {
	defaultBranch, err := s.DefaultBranch()
	if err != nil {
		return false, err
	}
	return branch == defaultBranch, nil
}

func (s *shellGit) GetFirstCommit() (string, error) {
	out, err := s.run("rev-list", "--max-parents=0", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get first commit: %w", err)
	}
	return strings.TrimSpace(out), nil
}

func (g *shellGit) RunInteractive(cmd string, args ...string) error {
	cmdArgs := append([]string{cmd}, args...)
	return g.runInteractive(cmdArgs...)
}

func (s *shellGit) IsPathStaged(path string) (bool, error) {
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

func (s *shellGit) StageAllExcept(excludePaths []string) error {
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
