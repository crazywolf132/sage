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
	configs := []struct {
		key, value string
	}{
		{"user.name", "Test User"},
		{"user.email", "test@example.com"},
		{"core.editor", "true"},
		{"core.autocrlf", "false"},
		{"core.mergeoptions", "--no-edit"},
		{"pull.rebase", "false"},
		{"advice.detachedHead", "false"},
	}

	for _, cfg := range configs {
		if err := git.RunInteractive("config", cfg.key, cfg.value); err != nil {
			t.Fatal(err)
		}
	}

	// Create initial commit
	if err := git.RunInteractive("commit", "--allow-empty", "-m", "Initial commit"); err != nil {
		t.Fatal(err)
	}

	// Set up main branch
	if err := git.RunInteractive("branch", "-M", "main"); err != nil {
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

func TestValidationFunctions(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Valid reference", "feature/branch", false},
		{"Empty reference", "", true},
		{"Command injection &&", "branch&&ls", true},
		{"Command injection ;", "branch;ls", true},
		{"Command injection |", "branch|ls", true},
		{"Command injection $", "branch$PATH", true},
		{"Command injection backtick", "branch`ls`", true},
		{"Command injection parentheses", "branch(ls)", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRef(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRef() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMockGitErrorHandling(t *testing.T) {
	mock := NewMockGit()

	// Test non-existent branch operations
	t.Run("NonExistentBranch", func(t *testing.T) {
		if err := mock.Checkout("non-existent"); err == nil {
			t.Error("Checkout() should error on non-existent branch")
		}

		if err := mock.DeleteBranch("non-existent"); err == nil {
			t.Error("DeleteBranch() should error on non-existent branch")
		}

		if err := mock.Push("non-existent", ""); err == nil {
			t.Error("Push() should error on non-existent branch")
		}
	})

	// Test stash operations with empty stash
	t.Run("EmptyStash", func(t *testing.T) {
		if err := mock.StashPop(); err == nil {
			t.Error("StashPop() should error on empty stash")
		}

		stashes, err := mock.StashList()
		if err != nil {
			t.Errorf("StashList() error = %v", err)
		}
		if len(stashes) != 0 {
			t.Error("StashList() should return empty list for new repo")
		}
	})

	// Test commit without changes
	t.Run("CommitWithoutChanges", func(t *testing.T) {
		mock.SetClean(true)
		if err := mock.Commit("empty commit", false, false); err == nil {
			t.Error("Commit() should error when no changes and allowEmpty=false")
		}
	})
}

func TestComplexGitOperations(t *testing.T) {
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

			// Test branch divergence tracking
			t.Run("BranchDivergence", func(t *testing.T) {
				if impl.name == "ShellGit" {
					// Create feature branch
					if err := impl.git.CreateBranch("feature"); err != nil {
						t.Fatal(err)
					}
					if err := impl.git.Checkout("feature"); err != nil {
						t.Fatal(err)
					}

					// Make changes on feature branch
					repo.createFile("feature.txt", "feature content")
					impl.git.StageAll()
					impl.git.Commit("feature commit", false, true)

					// Switch back to main and make different changes
					if err := impl.git.Checkout("main"); err != nil {
						t.Fatal(err)
					}
					repo.createFile("main.txt", "main content")
					impl.git.StageAll()
					impl.git.Commit("main commit", false, true)

					// Check divergence
					count, err := impl.git.GetCommitCount("main...feature")
					if err != nil {
						t.Fatal(err)
					}
					if count < 2 {
						t.Errorf("GetCommitCount(main...feature) = %v, want >= 2", count)
					}
				}
			})

			// Test complex merge scenarios
			t.Run("MergeScenarios", func(t *testing.T) {
				if impl.name == "ShellGit" {
					// Create and switch to feature branch
					if err := impl.git.CreateBranch("feature-merge"); err != nil {
						t.Fatal(err)
					}
					if err := impl.git.Checkout("feature-merge"); err != nil {
						t.Fatal(err)
					}

					// Create a file that will conflict
					repo.createFile("conflict.txt", "feature content")
					impl.git.StageAll()
					impl.git.Commit("feature change", false, true)

					// Switch to main and make conflicting change
					if err := impl.git.Checkout("main"); err != nil {
						t.Fatal(err)
					}
					repo.createFile("conflict.txt", "main content")
					impl.git.StageAll()
					impl.git.Commit("main change", false, true)

					// Try to merge and expect conflict
					err := impl.git.Merge("feature-merge")
					if err == nil {
						t.Error("Merge() should fail due to conflicts")
					}

					// Verify we're in a merging state
					isMerging, err := impl.git.IsMerging()
					if err != nil {
						t.Fatal(err)
					}
					if !isMerging {
						t.Error("IsMerging() = false, want true")
					}

					// Abort the merge
					if err := impl.git.MergeAbort(); err != nil {
						t.Fatal(err)
					}
				}
			})

			// Test complex stash operations
			t.Run("StashOperations", func(t *testing.T) {
				if impl.name == "ShellGit" {
					// Create and modify a file
					repo.createFile("stash1.txt", "stash1 content")
					if err := impl.git.StageAll(); err != nil {
						t.Fatal(err)
					}
					if err := impl.git.Commit("add stash1.txt", false, true); err != nil {
						t.Fatal(err)
					}

					// Modify the file
					repo.createFile("stash1.txt", "stash1 modified content")
					if err := impl.git.Stash("stash 1"); err != nil {
						t.Fatal(err)
					}

					// Create and modify another file
					repo.createFile("stash2.txt", "stash2 content")
					if err := impl.git.StageAll(); err != nil {
						t.Fatal(err)
					}
					if err := impl.git.Commit("add stash2.txt", false, true); err != nil {
						t.Fatal(err)
					}
					repo.createFile("stash2.txt", "stash2 modified content")
					if err := impl.git.Stash("stash 2"); err != nil {
						t.Fatal(err)
					}

					// Check stash list
					stashes, err := impl.git.StashList()
					if err != nil {
						t.Fatal(err)
					}
					if len(stashes) == 0 {
						t.Error("StashList() returned empty list, want stashes")
					}

					// Pop a stash and verify changes are restored
					if err := impl.git.StashPop(); err != nil {
						t.Fatal(err)
					}

					// Verify the file content
					content, err := os.ReadFile("stash2.txt")
					if err != nil {
						t.Fatal(err)
					}
					if string(content) != "stash2 modified content" {
						t.Errorf("Stashed file content = %q, want %q", string(content), "stash2 modified content")
					}

					// Check remaining stashes
					stashes, err = impl.git.StashList()
					if err != nil {
						t.Fatal(err)
					}
					if len(stashes) == 0 {
						t.Error("StashList() returned empty list, want 1 remaining stash")
					}
				}
			})
		})
	}
}

func TestPathValidation(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"Valid path", "file.txt", false},
		{"Valid nested path", "dir/file.txt", false},
		{"Empty path", "", true},
		{"Path traversal", "../file.txt", true},
		{"Command injection &&", "file&&ls", true},
		{"Command injection ;", "file;ls", true},
		{"Command injection |", "file|ls", true},
		{"Command injection $", "file$PATH", true},
		{"Command injection backtick", "file`ls`", true},
		{"Command injection parentheses", "file(ls)", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRebaseOperations(t *testing.T) {
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

				// Configure git to use a non-interactive editor
				if err := impl.git.RunInteractive("config", "core.editor", "true"); err != nil {
					t.Fatal(err)
				}
			}

			// Test rebase workflow
			t.Run("RebaseWorkflow", func(t *testing.T) {
				if impl.name == "ShellGit" {
					// Create and switch to feature branch
					if err := impl.git.CreateBranch("feature-rebase"); err != nil {
						t.Fatal(err)
					}
					if err := impl.git.Checkout("feature-rebase"); err != nil {
						t.Fatal(err)
					}

					// Make changes on feature branch
					repo.createFile("rebase.txt", "feature content")
					impl.git.StageAll()
					impl.git.Commit("feature change", false, true)

					// Switch to main and make changes that will conflict
					if err := impl.git.Checkout("main"); err != nil {
						t.Fatal(err)
					}
					repo.createFile("rebase.txt", "main content")
					impl.git.StageAll()
					impl.git.Commit("main change", false, true)

					// Switch back to feature and try to rebase
					if err := impl.git.Checkout("feature-rebase"); err != nil {
						t.Fatal(err)
					}

					// Attempt rebase (should fail due to conflict)
					err := impl.git.RunInteractive("rebase", "main")
					if err == nil {
						t.Error("Rebase should fail due to conflicts")
					}

					// Check if we're in rebase state
					isRebasing, err := impl.git.IsRebasing()
					if err != nil {
						t.Fatal(err)
					}
					if !isRebasing {
						t.Error("IsRebasing() = false during rebase")
					}

					// Abort the rebase
					if err := impl.git.RebaseAbort(); err != nil {
						t.Fatal(err)
					}

					// Verify we're no longer rebasing
					isRebasing, err = impl.git.IsRebasing()
					if err != nil {
						t.Fatal(err)
					}
					if isRebasing {
						t.Error("IsRebasing() = true after abort")
					}
				}
			})
		})
	}
}

func TestResetOperations(t *testing.T) {
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

			// Test reset workflow
			t.Run("ResetWorkflow", func(t *testing.T) {
				if impl.name == "ShellGit" {
					// Create initial state
					repo.createFile("reset.txt", "initial content")
					impl.git.StageAll()
					impl.git.Commit("initial commit", false, true)

					// Get the commit hash
					firstHash, err := impl.git.GetCommitHash("HEAD")
					if err != nil {
						t.Fatal(err)
					}

					// Make changes and commit
					repo.createFile("reset.txt", "modified content")
					impl.git.StageAll()
					impl.git.Commit("second commit", false, true)

					// Perform soft reset
					if err := impl.git.ResetSoft(firstHash); err != nil {
						t.Fatal(err)
					}

					// Verify the changes are staged
					isClean, err := impl.git.IsClean()
					if err != nil {
						t.Fatal(err)
					}
					if isClean {
						t.Error("IsClean() = true after soft reset, want false")
					}

					isStaged, err := impl.git.IsPathStaged("reset.txt")
					if err != nil {
						t.Fatal(err)
					}
					if !isStaged {
						t.Error("IsPathStaged() = false after soft reset, want true")
					}
				}
			})
		})
	}
}

func TestAdvancedGitOperations(t *testing.T) {
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

				// Configure git to use a non-interactive editor
				if err := impl.git.RunInteractive("config", "core.editor", "true"); err != nil {
					t.Fatal(err)
				}
			}

			// Test ancestor relationships
			t.Run("AncestorRelationships", func(t *testing.T) {
				if impl.name == "ShellGit" {
					// Create a linear history
					repo.createFile("ancestor.txt", "initial")
					impl.git.StageAll()
					impl.git.Commit("first", false, true)
					firstHash, err := impl.git.GetCommitHash("HEAD")
					if err != nil {
						t.Fatal(err)
					}

					repo.createFile("ancestor.txt", "second")
					impl.git.StageAll()
					impl.git.Commit("second", false, true)
					secondHash, err := impl.git.GetCommitHash("HEAD")
					if err != nil {
						t.Fatal(err)
					}

					// Check ancestor relationship
					isAncestor, err := impl.git.IsAncestor(firstHash, secondHash)
					if err != nil {
						t.Fatal(err)
					}
					if !isAncestor {
						t.Error("IsAncestor() = false, want true for linear history")
					}
				}
			})

			// Test advanced log operations
			t.Run("AdvancedLogOperations", func(t *testing.T) {
				if impl.name == "ShellGit" {
					// Create some commits with stats
					repo.createFile("file1.txt", "content1")
					impl.git.StageAll()
					impl.git.Commit("add file1", false, true)

					repo.createFile("file2.txt", "content2")
					impl.git.StageAll()
					impl.git.Commit("add file2", false, true)

					// Get log with stats
					log, err := impl.git.Log("", 2, true, false)
					if err != nil {
						t.Fatal(err)
					}

					// Verify log contains commit info and stats
					if !strings.Contains(log, "add file1") || !strings.Contains(log, "add file2") {
						t.Error("Log() missing commit messages")
					}
				}
			})

			// Test selective staging
			t.Run("SelectiveStaging", func(t *testing.T) {
				if impl.name == "ShellGit" {
					// Create multiple files
					repo.createFile("include.txt", "include")
					repo.createFile("exclude.txt", "exclude")

					// Stage all except excluded file
					if err := impl.git.StageAll(); err != nil {
						t.Fatal(err)
					}

					// Unstage the excluded file
					if err := impl.git.RunInteractive("reset", "HEAD", "exclude.txt"); err != nil {
						t.Fatal(err)
					}

					// Get status to verify staging
					status, err := impl.git.StatusPorcelain()
					if err != nil {
						t.Fatal(err)
					}

					// Parse status to check staging
					lines := strings.Split(strings.TrimSpace(status), "\n")
					includeStaged := false
					excludeUnstaged := false
					for _, line := range lines {
						if strings.HasSuffix(line, "include.txt") && strings.HasPrefix(line, "A") {
							includeStaged = true
						}
						if strings.HasSuffix(line, "exclude.txt") && strings.HasPrefix(line, "??") {
							excludeUnstaged = true
						}
					}

					if !includeStaged {
						t.Error("include.txt should be staged")
					}
					if !excludeUnstaged {
						t.Error("exclude.txt should be unstaged")
					}
				}
			})
		})
	}
}
