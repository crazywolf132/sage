package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testRepo is a helper struct to manage a test Git repository
type testRepo struct {
	path string
	t    *testing.T
	git  Service
}

// newTestRepo creates a new Git repository for testing
func newTestRepo(t *testing.T) *testRepo {
	t.Helper()

	// Create temporary directory
	dir, err := os.MkdirTemp("", "sage-git-test")
	if err != nil {
		t.Fatal(err)
	}

	// Initialize Git repo
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Initialize Git repo
	git := NewShellGit()
	if err := git.RunInteractive("init"); err != nil {
		t.Fatal(err)
	}

	// Configure Git for tests
	if err := git.RunInteractive("config", "user.name", "Test User"); err != nil {
		t.Fatal(err)
	}
	if err := git.RunInteractive("config", "user.email", "test@example.com"); err != nil {
		t.Fatal(err)
	}

	// Create initial commit
	if err := git.RunInteractive("commit", "--allow-empty", "-m", "Initial commit"); err != nil {
		t.Fatal(err)
	}

	return &testRepo{
		path: dir,
		t:    t,
		git:  git,
	}
}

// cleanup removes the test repository
func (r *testRepo) cleanup() {
	if err := os.RemoveAll(r.path); err != nil {
		r.t.Errorf("Failed to cleanup test repo: %v", err)
	}
}

// createFile creates a file with the given content
func (r *testRepo) createFile(name, content string) {
	r.t.Helper()
	path := filepath.Join(r.path, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		r.t.Fatal(err)
	}
}

func TestGitOperations(t *testing.T) {
	// Test both mock and real implementations
	implementations := []struct {
		name string
		git  Service
	}{
		{"MockGit", NewMockGit()},
		{"ShellGit", NewShellGit()},
	}

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			var repo *testRepo
			if impl.name == "ShellGit" {
				repo = newTestRepo(t)
				defer repo.cleanup()
				impl.git = repo.git
			}

			// Test IsRepo
			t.Run("IsRepo", func(t *testing.T) {
				isRepo, err := impl.git.IsRepo()
				if err != nil {
					t.Errorf("IsRepo() error = %v", err)
				}
				if !isRepo {
					t.Error("IsRepo() = false, want true")
				}
			})

			// Test IsClean
			t.Run("IsClean", func(t *testing.T) {
				isClean, err := impl.git.IsClean()
				if err != nil {
					t.Errorf("IsClean() error = %v", err)
				}
				if !isClean {
					t.Error("IsClean() = false, want true")
				}

				if impl.name == "ShellGit" {
					// Create a file to make repo dirty
					repo.createFile("test.txt", "test content")
					isClean, err = impl.git.IsClean()
					if err != nil {
						t.Errorf("IsClean() error = %v", err)
					}
					if isClean {
						t.Error("IsClean() = true, want false")
					}
				}
			})

			// Test StageAll and Commit
			t.Run("StageAndCommit", func(t *testing.T) {
				if impl.name == "ShellGit" {
					repo.createFile("test.txt", "test content")
				}

				if err := impl.git.StageAll(); err != nil {
					t.Errorf("StageAll() error = %v", err)
				}

				if err := impl.git.Commit("test commit", false, true); err != nil {
					t.Errorf("Commit() error = %v", err)
				}

				// Verify repo is clean after commit
				isClean, err := impl.git.IsClean()
				if err != nil {
					t.Errorf("IsClean() error = %v", err)
				}
				if !isClean {
					t.Error("IsClean() = false, want true after commit")
				}
			})

			// Test Branch Operations
			t.Run("BranchOperations", func(t *testing.T) {
				// Get current branch
				branch, err := impl.git.CurrentBranch()
				if err != nil {
					t.Errorf("CurrentBranch() error = %v", err)
				}
				if branch == "" {
					t.Error("CurrentBranch() returned empty string")
				}

				// Create new branch
				newBranch := "test-branch"
				if err := impl.git.CreateBranch(newBranch); err != nil {
					t.Errorf("CreateBranch() error = %v", err)
				}

				// Checkout new branch
				if err := impl.git.Checkout(newBranch); err != nil {
					t.Errorf("Checkout() error = %v", err)
				}

				// Verify current branch
				currentBranch, err := impl.git.CurrentBranch()
				if err != nil {
					t.Errorf("CurrentBranch() error = %v", err)
				}
				if currentBranch != newBranch {
					t.Errorf("CurrentBranch() = %v, want %v", currentBranch, newBranch)
				}

				// List branches
				branches, err := impl.git.ListBranches()
				if err != nil {
					t.Errorf("ListBranches() error = %v", err)
				}
				if len(branches) < 2 { // Should have at least main/master and test-branch
					t.Errorf("ListBranches() returned %d branches, want at least 2", len(branches))
				}

				// Delete branch (switch back to main first)
				if err := impl.git.Checkout("main"); err != nil {
					if err := impl.git.Checkout("master"); err != nil {
						t.Errorf("Checkout(main/master) error = %v", err)
					}
				}
				if err := impl.git.DeleteBranch(newBranch); err != nil {
					t.Errorf("DeleteBranch() error = %v", err)
				}
			})

			// Test Diff Operations
			t.Run("DiffOperations", func(t *testing.T) {
				if impl.name == "ShellGit" {
					repo.createFile("diff-test.txt", "initial content")
					impl.git.StageAll()
					impl.git.Commit("add test file", false, true)

					// Modify file
					repo.createFile("diff-test.txt", "modified content")
				}

				diff, err := impl.git.GetDiff()
				if err != nil {
					t.Errorf("GetDiff() error = %v", err)
				}
				if impl.name == "ShellGit" && !strings.Contains(diff, "diff --git") {
					t.Error("GetDiff() did not return a valid diff")
				}
			})

			// Test Stash Operations
			t.Run("StashOperations", func(t *testing.T) {
				if impl.name == "ShellGit" {
					repo.createFile("stash-test.txt", "stash content")
				}

				if err := impl.git.Stash("test stash"); err != nil {
					t.Errorf("Stash() error = %v", err)
				}

				stashes, err := impl.git.StashList()
				if err != nil {
					t.Errorf("StashList() error = %v", err)
				}
				if len(stashes) == 0 {
					t.Error("StashList() returned empty list after stash")
				}

				if err := impl.git.StashPop(); err != nil {
					t.Errorf("StashPop() error = %v", err)
				}
			})

			// Test Status Operations
			t.Run("StatusOperations", func(t *testing.T) {
				if impl.name == "ShellGit" {
					repo.createFile("status-test.txt", "status content")
				}

				status, err := impl.git.StatusPorcelain()
				if err != nil {
					t.Errorf("StatusPorcelain() error = %v", err)
				}
				if impl.name == "ShellGit" && !strings.Contains(status, "??") {
					t.Error("StatusPorcelain() did not show untracked file")
				}
			})

			// Test Log Operations
			t.Run("LogOperations", func(t *testing.T) {
				if impl.name == "ShellGit" {
					repo.createFile("log-test.txt", "log content")
					impl.git.StageAll()
					impl.git.Commit("test log", false, true)
				}

				log, err := impl.git.Log("", 1, true, false)
				if err != nil {
					t.Errorf("Log() error = %v", err)
				}
				if impl.name == "ShellGit" && !strings.Contains(log, "test log") {
					t.Error("Log() did not return commit message")
				}
			})
		})
	}
}

func TestMockGitSpecific(t *testing.T) {
	mock := NewMockGit()

	// Test call tracking
	mock.IsRepo()
	if count := mock.GetCallCount("IsRepo"); count != 1 {
		t.Errorf("GetCallCount(IsRepo) = %v, want 1", count)
	}

	// Test state manipulation
	mock.SetClean(false)
	isClean, _ := mock.IsClean()
	if isClean {
		t.Error("IsClean() = true after SetClean(false)")
	}

	// Test branch management
	mock.SetCurrentBranch("test")
	branch, _ := mock.CurrentBranch()
	if branch != "test" {
		t.Errorf("CurrentBranch() = %v, want test", branch)
	}

	mock.AddBranch("feature")
	branches, _ := mock.ListBranches()
	found := false
	for _, b := range branches {
		if b == "feature" {
			found = true
			break
		}
	}
	if !found {
		t.Error("AddBranch() did not add branch")
	}
}
