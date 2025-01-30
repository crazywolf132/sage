package gitutils

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/viper"
)

// DefaultRunner is the Git runner implementation to use
var DefaultRunner GitRunner = &RealGitRunner{}

// RealGitRunner implements GitRunner interface with actual Git commands
type RealGitRunner struct{}

// RunGitCommand runs "git <args...>" and returns an error if it fails.
func (g *RealGitRunner) RunGitCommand(args ...string) error {
	// If "explain mode" is enabled, show the command
	if viper.GetBool("sageExplain") {
		fmt.Printf("[explain] Running: git %s\n", strings.Join(args, " "))
	}

	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// RunGitCommand is a package-level function that delegates to DefaultRunner
func RunGitCommand(args ...string) error {
	return DefaultRunner.RunGitCommand(args...)
}

// IsWorkingDirectoryClean checks if there are no uncommitted changes.
func (g *RealGitRunner) IsWorkingDirectoryClean() (bool, error) {
	var stdout bytes.Buffer
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Stderr = os.Stderr // If there's an error, let's show it
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return false, err
	}
	output := strings.TrimSpace(stdout.String())
	return (output == ""), nil
}

// IsWorkingDirectoryClean is a package-level function that delegates to DefaultRunner
func IsWorkingDirectoryClean() (bool, error) {
	return DefaultRunner.IsWorkingDirectoryClean()
}

// GetCurrentBranch returns the name of the currently checked-out Git branch.
func (g *RealGitRunner) GetCurrentBranch() (string, error) {
	var stdout bytes.Buffer
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Stderr = os.Stderr
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}

// GetCurrentBranch is a package-level function that delegates to DefaultRunner
func GetCurrentBranch() (string, error) {
	return DefaultRunner.GetCurrentBranch()
}

// IsMergeInProgress checks if there's a MERGE_HEAD indicating an ongoing merge.
func (g *RealGitRunner) IsMergeInProgress() (bool, error) {
	mergeHead := ".git/MERGE_HEAD"
	if _, err := os.Stat(mergeHead); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, fmt.Errorf("error checking merge state: %w", err)
	}
}

// IsMergeInProgress is a package-level function that delegates to DefaultRunner
func IsMergeInProgress() (bool, error) {
	return DefaultRunner.IsMergeInProgress()
}

// IsRebaseInProgress checks if there's a REBASE_HEAD indicating an ongoing rebase.
func (g *RealGitRunner) IsRebaseInProgress() (bool, error) {
	rebaseHead := ".git/REBASE_HEAD"
	if _, err := os.Stat(rebaseHead); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, fmt.Errorf("error checking rebase state: %w", err)
	}
}

// IsRebaseInProgress is a package-level function that delegates to DefaultRunner
func IsRebaseInProgress() (bool, error) {
	return DefaultRunner.IsRebaseInProgress()
}

// RunGitCommandWithOutput runs a git command and returns its output as a string
func (g *RealGitRunner) RunGitCommandWithOutput(args ...string) (string, error) {
	var stdout bytes.Buffer
	if viper.GetBool("sageExplain") {
		fmt.Printf("[explain] Running: git %s\n", strings.Join(args, " "))
	}

	cmd := exec.Command("git", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}

// RunGitCommandWithOutput is a package-level function that delegates to DefaultRunner
func RunGitCommandWithOutput(args ...string) (string, error) {
	return DefaultRunner.RunGitCommandWithOutput(args...)
}

// GetBranches returns a list of all local branches
func (g *RealGitRunner) GetBranches() ([]string, error) {
	output, err := g.RunGitCommandWithOutput("branch", "--list", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}

	if output == "" {
		return []string{}, nil
	}

	branches := strings.Split(output, "\n")
	return branches, nil
}

// GetBranches is a package-level function that delegates to DefaultRunner
func GetBranches() ([]string, error) {
	return DefaultRunner.GetBranches()
}

// BranchExists checks if a branch exists
func (g *RealGitRunner) BranchExists(branchName string) (bool, error) {
	output, err := g.RunGitCommandWithOutput("branch", "--list", branchName)
	if err != nil {
		return false, err
	}
	return output != "", nil
}

// BranchExists is a package-level function that delegates to DefaultRunner
func BranchExists(branchName string) (bool, error) {
	return DefaultRunner.BranchExists(branchName)
}

// IsWorkingTreeClean checks if the git working tree is clean (no uncommitted changes)
func IsWorkingTreeClean() (bool, error) {
	// Check for staged and unstaged changes
	output, err := DefaultRunner.RunGitCommandWithOutput("status", "--porcelain")
	if err != nil {
		return false, err
	}

	return output == "", nil
}

func GetDefaultBranch() (string, error) {
	// Check remote HEAD
	if out, err := RunGitCommandWithOutput("symbolic-ref", "refs/remotes/origin/HEAD"); err == nil {
		parts := strings.Split(strings.TrimSpace(out), "/")
		return parts[len(parts)-1], nil
	}

	// Fallback to checking common branch names
	if err := RunGitCommand("show-ref", "--verify", "refs/heads/main"); err == nil {
		return "main", nil
	}
	if err := RunGitCommand("show-ref", "--verify", "refs/heads/master"); err == nil {
		return "master", nil
	}

	return "", fmt.Errorf("could not determine default branch")
}

func GetMergedBranches(target string) ([]string, error) {
	out, err := RunGitCommandWithOutput("branch", "--merged", target)
	if err != nil {
		return nil, err
	}

	var branches []string
	for _, line := range strings.Split(out, "\n") {
		branch := strings.TrimSpace(strings.TrimPrefix(line, "* "))
		if branch != "" {
			branches = append(branches, branch)
		}
	}
	return branches, nil
}
