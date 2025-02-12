package app

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/crazywolf132/sage/internal/gh"
	"github.com/crazywolf132/sage/internal/git"
)

// mockGit is a test implementation of git.Service that records operations
type mockGit struct {
	git.Service
	isRepo        bool
	isClean       bool
	currentBranch string
	status        string
	branches      []string
	defaultBranch string
	operations    []string
	err           error
	isMerging     bool
	isRebasing    bool
	staged        map[string]bool
	ops           []string
}

// mockGHClient is a test implementation of gh.Client
type mockGHClient struct {
	gh.Client
	prTemplate string
	pr         *gh.PullRequest
	err        error
}

func newMockGit() *mockGit {
	return &mockGit{
		isRepo:        true,
		currentBranch: "feature",
		defaultBranch: "main",
		status:        "clean",
		ops:           []string{},
		staged:        make(map[string]bool),
	}
}

func (m *mockGit) IsRepo() (bool, error) {
	m.operations = append(m.operations, "IsRepo")
	if !m.isRepo {
		return false, fmt.Errorf("not a git repository")
	}
	return m.isRepo, m.err
}

func (m *mockGit) IsClean() (bool, error) {
	m.operations = append(m.operations, "IsClean")
	return m.isClean, m.err
}

func (m *mockGit) CurrentBranch() (string, error) {
	m.operations = append(m.operations, "CurrentBranch")
	return m.currentBranch, m.err
}

func (m *mockGit) StatusPorcelain() (string, error) {
	m.operations = append(m.operations, "StatusPorcelain")
	return m.status, m.err
}

func (m *mockGit) ListBranches() ([]string, error) {
	m.operations = append(m.operations, "ListBranches")
	return m.branches, m.err
}

func (m *mockGit) DefaultBranch() (string, error) {
	m.operations = append(m.operations, "DefaultBranch")
	return m.defaultBranch, m.err
}

func (m *mockGit) Checkout(branch string) error {
	m.operations = append(m.operations, "Checkout:"+branch)
	return m.err
}

func (m *mockGit) Push(branch string, forceType string) error {
	m.operations = append(m.operations, "Push:"+branch)
	return m.err
}

func (m *mockGit) CreateBranch(name string) error {
	m.operations = append(m.operations, "CreateBranch:"+name)
	return m.err
}

func (m *mockGit) Pull() error {
	m.operations = append(m.operations, "Pull")
	return m.err
}

func (m *mockGit) FetchAll() error {
	m.operations = append(m.operations, "FetchAll")
	return m.err
}

func (m *mockGit) Commit(msg string, allowEmpty bool, stageAll bool) error {
	m.operations = append(m.operations, fmt.Sprintf("Commit:%s,empty=%v,stage=%v", msg, allowEmpty, stageAll))
	return m.err
}

func (m *mockGit) Clean() error {
	m.operations = append(m.operations, "Clean")
	return m.err
}

func (m *mockGit) MergedBranches(base string) ([]string, error) {
	m.operations = append(m.operations, "MergedBranches")
	return []string{}, m.err
}

func (m *mockGit) IsMerging() (bool, error) {
	m.operations = append(m.operations, "IsMerging")
	return m.isMerging, m.err
}

func (m *mockGit) IsRebasing() (bool, error) {
	m.operations = append(m.operations, "IsRebasing")
	return m.isRebasing, m.err
}

func (m *mockGit) MergeAbort() error {
	m.operations = append(m.operations, "MergeAbort")
	m.isMerging = false
	return m.err
}

func (m *mockGit) RebaseAbort() error {
	m.operations = append(m.operations, "RebaseAbort")
	m.isRebasing = false
	return m.err
}

func (m *mockGit) PullFF() error {
	m.operations = append(m.operations, "PullFF")
	return m.err
}

func (m *mockGit) PullRebase() error {
	m.operations = append(m.operations, "PullRebase")
	return m.err
}

func (m *mockGit) GetCommitHash(ref string) (string, error) {
	m.operations = append(m.operations, "GetCommitHash:"+ref)
	return "mock-hash", m.err
}

func (m *mockGit) Stash(message string) error {
	m.operations = append(m.operations, "Stash:"+message)
	return m.err
}

func (m *mockGit) StashPop() error {
	m.operations = append(m.operations, "StashPop")
	return m.err
}

func (m *mockGit) StashList() ([]string, error) {
	m.operations = append(m.operations, "StashList")
	return []string{}, m.err
}

func (m *mockGit) GetMergeBase(branch1, branch2 string) (string, error) {
	m.operations = append(m.operations, fmt.Sprintf("GetMergeBase:%s,%s", branch1, branch2))
	return "mock-merge-base", m.err
}

func (m *mockGit) GetCommitCount(revisionRange string) (int, error) {
	m.operations = append(m.operations, "GetCommitCount:"+revisionRange)
	return 1, m.err
}

func (m *mockGit) GetBranchDivergence(branch1, branch2 string) (int, error) {
	m.operations = append(m.operations, fmt.Sprintf("GetBranchDivergence:%s,%s", branch1, branch2))
	return 1, m.err
}

func (m *mockGit) IsAncestor(commit1, commit2 string) (bool, error) {
	m.operations = append(m.operations, fmt.Sprintf("IsAncestor:%s,%s", commit1, commit2))
	return true, m.err
}

func (m *mockGit) GetDiff() (string, error) {
	m.operations = append(m.operations, "GetDiff")
	return "", m.err
}

func (m *mockGit) Log(branch string, limit int, stats, all bool) (string, error) {
	m.operations = append(m.operations, fmt.Sprintf("Log:%s,limit=%d,stats=%v,all=%v", branch, limit, stats, all))
	return "", m.err
}

func (m *mockGit) SquashCommits(startCommit string) error {
	m.operations = append(m.operations, "SquashCommits:"+startCommit)
	return m.err
}

func (m *mockGit) IsHeadBranch(branch string) (bool, error) {
	m.operations = append(m.operations, "IsHeadBranch:"+branch)
	return false, m.err
}

func (m *mockGit) GetFirstCommit() (string, error) {
	m.operations = append(m.operations, "GetFirstCommit")
	return "mock-first-commit", m.err
}

func (m *mockGit) RunInteractive(cmd string, args ...string) error {
	m.operations = append(m.operations, fmt.Sprintf("RunInteractive:%s,%s", cmd, strings.Join(args, ",")))
	return m.err
}

func (m *mockGit) GetBranchLastCommit(branch string) (time.Time, error) {
	m.operations = append(m.operations, "GetBranchLastCommit:"+branch)
	return time.Now(), m.err
}

func (m *mockGit) GetBranchCommitCount(branch string) (int, error) {
	m.operations = append(m.operations, "GetBranchCommitCount:"+branch)
	return 1, m.err
}

func (m *mockGit) GetBranchMergeConflicts(branch string) (int, error) {
	m.operations = append(m.operations, "GetBranchMergeConflicts:"+branch)
	return 0, m.err
}

func (m *mockGit) StageAll() error {
	m.operations = append(m.operations, "StageAll")
	m.staged["mock-file"] = true
	return m.err
}

func (m *mockGit) withIsMerging(merging bool) *mockGit {
	m.isMerging = merging
	return m
}

func (m *mockGit) withIsRebasing(rebasing bool) *mockGit {
	m.isRebasing = rebasing
	return m
}

// Test cases

func TestStartBranch(t *testing.T) {
	tests := []struct {
		name       string
		mock       *mockGit
		branchName string
		push       bool
		wantErr    bool
		wantOps    []string
	}{
		{
			name: "successful branch creation",
			mock: func() *mockGit {
				m := newMockGit()
				m.isRepo = true
				m.defaultBranch = "main"
				return m
			}(),
			branchName: "feature/test",
			push:       false,
			wantOps: []string{
				"IsRepo",
				"DefaultBranch",
				"FetchAll",
				"Checkout:main",
				"Pull",
				"CreateBranch:feature/test",
				"Checkout:feature/test",
			},
		},
		{
			name: "successful branch creation with push",
			mock: func() *mockGit {
				m := newMockGit()
				m.isRepo = true
				m.defaultBranch = "main"
				return m
			}(),
			branchName: "feature/test",
			push:       true,
			wantOps: []string{
				"IsRepo",
				"DefaultBranch",
				"FetchAll",
				"Checkout:main",
				"Pull",
				"CreateBranch:feature/test",
				"Checkout:feature/test",
				"Push:feature/test",
			},
		},
		{
			name: "not a git repo",
			mock: func() *mockGit {
				m := newMockGit()
				m.isRepo = false
				return m
			}(),
			branchName: "feature/test",
			wantErr:    true,
			wantOps:    []string{"IsRepo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := StartBranch(tt.mock, tt.branchName, tt.push)
			if (err != nil) != tt.wantErr {
				t.Errorf("StartBranch() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !operationsMatch(tt.mock.operations, tt.wantOps) {
				t.Errorf("StartBranch() operations = %v, want %v", tt.mock.operations, tt.wantOps)
			}
		})
	}
}

func TestSwitchBranch(t *testing.T) {
	tests := []struct {
		name       string
		mock       *mockGit
		branchName string
		wantErr    bool
		wantOps    []string
	}{
		{
			name: "successful branch switch",
			mock: func() *mockGit {
				m := newMockGit()
				m.isRepo = true
				return m
			}(),
			branchName: "feature/test",
			wantOps: []string{
				"IsRepo",
				"Checkout:feature/test",
			},
		},
		{
			name: "not a git repo",
			mock: func() *mockGit {
				m := newMockGit()
				m.isRepo = false
				return m
			}(),
			branchName: "feature/test",
			wantErr:    true,
			wantOps:    []string{"IsRepo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SwitchBranch(tt.mock, tt.branchName)
			if (err != nil) != tt.wantErr {
				t.Errorf("SwitchBranch() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !operationsMatch(tt.mock.operations, tt.wantOps) {
				t.Errorf("SwitchBranch() operations = %v, want %v", tt.mock.operations, tt.wantOps)
			}
		})
	}
}

func TestGetRepoStatus(t *testing.T) {
	tests := []struct {
		name       string
		mock       *mockGit
		wantStatus *RepoStatus
		wantErr    bool
	}{
		{
			name: "clean repo",
			mock: func() *mockGit {
				m := newMockGit()
				m.isRepo = true
				m.currentBranch = "main"
				m.status = ""
				return m
			}(),
			wantStatus: &RepoStatus{
				Branch:  "main",
				Changes: nil,
			},
		},
		{
			name: "repo with changes",
			mock: func() *mockGit {
				m := newMockGit()
				m.isRepo = true
				m.currentBranch = "feature"
				m.status = "M  file1.txt\n?? file2.txt"
				return m
			}(),
			wantStatus: &RepoStatus{
				Branch: "feature",
				Changes: []FileChange{
					{Symbol: "M", File: "file1.txt", Description: "Staged Modified"},
					{Symbol: "?", File: "file2.txt", Description: "Untracked"},
				},
			},
		},
		{
			name: "not a git repo",
			mock: func() *mockGit {
				m := newMockGit()
				m.isRepo = false
				return m
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := GetRepoStatus(tt.mock)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRepoStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if status.Branch != tt.wantStatus.Branch {
					t.Errorf("GetRepoStatus() branch = %v, want %v", status.Branch, tt.wantStatus.Branch)
				}
				if len(status.Changes) != len(tt.wantStatus.Changes) {
					t.Errorf("GetRepoStatus() changes length = %v, want %v", len(status.Changes), len(tt.wantStatus.Changes))
				} else {
					for i, change := range status.Changes {
						if change.Symbol != tt.wantStatus.Changes[i].Symbol ||
							change.File != tt.wantStatus.Changes[i].File ||
							change.Description != tt.wantStatus.Changes[i].Description {
							t.Errorf("GetRepoStatus() change[%d] = %+v, want %+v", i, change, tt.wantStatus.Changes[i])
						}
					}
				}
			}
		})
	}
}

func TestPushCurrentBranch(t *testing.T) {
	tests := []struct {
		name    string
		mock    *mockGit
		force   bool
		wantErr bool
		wantOps []string
	}{
		{
			name: "successful push",
			mock: &mockGit{
				isRepo:        true,
				currentBranch: "feature",
			},
			force: false,
			wantOps: []string{
				"IsRepo",
				"CurrentBranch",
				"Push:feature",
			},
		},
		{
			name: "successful force push",
			mock: &mockGit{
				isRepo:        true,
				currentBranch: "feature",
			},
			force: true,
			wantOps: []string{
				"IsRepo",
				"CurrentBranch",
				"Push:feature",
			},
		},
		{
			name: "not a git repo",
			mock: &mockGit{
				isRepo: false,
			},
			wantErr: true,
			wantOps: []string{"IsRepo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PushCurrentBranch(tt.mock, tt.force)
			if (err != nil) != tt.wantErr {
				t.Errorf("PushCurrentBranch() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !operationsMatch(tt.mock.operations, tt.wantOps) {
				t.Errorf("PushCurrentBranch() operations = %v, want %v", tt.mock.operations, tt.wantOps)
			}
		})
	}
}

func TestCommit(t *testing.T) {
	tests := []struct {
		name    string
		mock    *mockGit
		opts    CommitOptions
		wantErr bool
		wantOps []string
	}{
		{
			name: "successful commit",
			mock: func() *mockGit {
				m := newMockGit()
				m.isRepo = true
				m.isClean = false
				m.status = "M  file1.txt"
				return m
			}(),
			opts: CommitOptions{
				Message: "test commit",
				UseAI:   false,
			},
			wantOps: []string{
				"IsRepo",
				"StatusPorcelain",
				"CurrentBranch",
				"StageAll",
				"Commit:test commit,empty=false,stage=true",
				"GetCommitHash:HEAD",
			},
		},
		{
			name: "clean repo",
			mock: func() *mockGit {
				m := newMockGit()
				m.isRepo = true
				m.isClean = true
				m.status = ""
				return m
			}(),
			opts: CommitOptions{
				Message: "test commit",
			},
			wantErr: true,
			wantOps: []string{
				"IsRepo",
				"StatusPorcelain",
			},
		},
		{
			name: "not a git repo",
			mock: func() *mockGit {
				m := newMockGit()
				m.isRepo = false
				return m
			}(),
			opts: CommitOptions{
				Message: "test commit",
			},
			wantErr: true,
			wantOps: []string{"IsRepo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Commit(tt.mock, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Commit() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if !operationsMatch(tt.mock.operations, tt.wantOps) {
					t.Errorf("Commit() operations = %v, want %v", tt.mock.operations, tt.wantOps)
				}
				if result.ActualMessage != tt.opts.Message {
					t.Errorf("Commit() message = %v, want %v", result.ActualMessage, tt.opts.Message)
				}
			}
		})
	}
}

func TestSyncBranch(t *testing.T) {
	tests := []struct {
		name    string
		mock    *mockGit
		abort   bool
		cont    bool
		wantErr bool
		wantOps []string
	}{
		{
			name:  "abort merge",
			mock:  newMockGit().withIsMerging(true),
			abort: true,
			wantOps: []string{
				"IsRepo",
				"IsMerging",
				"MergeAbort",
				"GetCommitHash:HEAD",
				"CurrentBranch",
				"DefaultBranch",
				"GetCommitHash:HEAD",
				"IsClean",
				"Stash:sage-sync-",
				"GetCommitHash:HEAD",
				"CurrentBranch",
				"Checkout:main",
				"PullFF",
				"Checkout:feature",
				"CurrentBranch",
				"Checkout:feature",
				"RunInteractive:rebase,--onto,main,main,feature",
				"GetCommitHash:HEAD",
				"CurrentBranch",
				"StashPop",
				"GetCommitHash:HEAD",
				"Push:feature",
			},
		},
		{
			name:  "abort rebase",
			mock:  newMockGit().withIsRebasing(true),
			abort: true,
			wantOps: []string{
				"IsRepo",
				"IsMerging",
				"IsRebasing",
				"RebaseAbort",
				"GetCommitHash:HEAD",
				"CurrentBranch",
				"DefaultBranch",
				"GetCommitHash:HEAD",
				"IsClean",
				"Stash:sage-sync-",
				"GetCommitHash:HEAD",
				"CurrentBranch",
				"Checkout:main",
				"PullFF",
				"Checkout:feature",
				"CurrentBranch",
				"Checkout:feature",
				"RunInteractive:rebase,--onto,main,main,feature",
				"GetCommitHash:HEAD",
				"CurrentBranch",
				"StashPop",
				"GetCommitHash:HEAD",
				"Push:feature",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SyncBranch(tt.mock, tt.abort, tt.cont)
			if (err != nil) != tt.wantErr {
				t.Errorf("SyncBranch() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				// Check if operations match, ignoring dynamic parts of stash messages
				gotOps := tt.mock.operations
				for i, op := range gotOps {
					if strings.HasPrefix(op, "Stash:sage-sync-") {
						gotOps[i] = "Stash:sage-sync-"
					}
				}
				if !operationsMatch(gotOps, tt.wantOps) {
					t.Errorf("SyncBranch() operations = %v, want %v", gotOps, tt.wantOps)
				}
			}
		})
	}
}

func TestFindCleanableBranches(t *testing.T) {
	tests := []struct {
		name    string
		mock    *mockGit
		wantErr bool
		wantOps []string
	}{
		{
			name: "successful clean",
			mock: func() *mockGit {
				m := newMockGit()
				m.isRepo = true
				return m
			}(),
			wantOps: []string{
				"IsRepo",
				"DefaultBranch",
				"MergedBranches",
				"CurrentBranch",
			},
		},
		{
			name: "not a git repo",
			mock: func() *mockGit {
				m := newMockGit()
				m.isRepo = false
				return m
			}(),
			wantErr: true,
			wantOps: []string{"IsRepo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FindCleanableBranches(tt.mock)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindCleanableBranches() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !operationsMatch(tt.mock.operations, tt.wantOps) {
				t.Errorf("FindCleanableBranches() operations = %v, want %v", tt.mock.operations, tt.wantOps)
			}
			if !tt.wantErr && result == nil {
				t.Error("FindCleanableBranches() result is nil")
			}
		})
	}
}

func TestCreatePullRequest(t *testing.T) {
	tests := []struct {
		name      string
		mock      *mockGit
		ghc       *mockGHClient
		opts      CreatePROpts
		wantErr   bool
		wantOps   []string
		wantPRNum int
	}{
		{
			name: "successful PR creation",
			mock: func() *mockGit {
				m := newMockGit()
				m.isRepo = true
				m.currentBranch = "feature"
				m.defaultBranch = "main"
				return m
			}(),
			ghc: &mockGHClient{
				pr: &gh.PullRequest{
					Number: 123,
					Title:  "Test PR",
				},
			},
			opts: CreatePROpts{
				Title:     "Test PR",
				Body:      "Test body",
				Base:      "", // Let it use default branch
				Draft:     false,
				Reviewers: []string{"reviewer1"},
				Labels:    []string{"bug"},
			},
			wantOps: []string{
				"IsRepo",
				"CurrentBranch",
				"Push:feature",
				"DefaultBranch",
			},
			wantPRNum: 123,
		},
		{
			name: "not a git repo",
			mock: func() *mockGit {
				m := newMockGit()
				m.isRepo = false
				return m
			}(),
			ghc: &mockGHClient{},
			opts: CreatePROpts{
				Title: "Test PR",
			},
			wantErr: true,
			wantOps: []string{"IsRepo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr, err := CreatePullRequest(tt.mock, tt.ghc, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreatePullRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if !operationsMatch(tt.mock.operations, tt.wantOps) {
					t.Errorf("CreatePullRequest() operations = %v, want %v", tt.mock.operations, tt.wantOps)
				}
				if pr.Number != tt.wantPRNum {
					t.Errorf("CreatePullRequest() PR number = %v, want %v", pr.Number, tt.wantPRNum)
				}
			}
		})
	}
}

func TestSyncBranchContinue(t *testing.T) {
	tests := []struct {
		name    string
		mock    *mockGit
		abort   bool
		cont    bool
		wantErr bool
		wantOps []string
	}{
		{
			name: "continue merge",
			mock: func() *mockGit {
				m := newMockGit()
				m.isRepo = true
				m.isMerging = true
				m.currentBranch = "feature"
				m.defaultBranch = "main"
				return m
			}(),
			cont: true,
			wantOps: []string{
				"IsRepo",
				"CurrentBranch",
				"DefaultBranch",
				"GetCommitHash:HEAD",
				"IsClean",
				"Stash:sage-sync-",
				"GetCommitHash:HEAD",
				"CurrentBranch",
				"Checkout:main",
				"PullFF",
				"Checkout:feature",
				"CurrentBranch",
				"Checkout:feature",
				"RunInteractive:rebase,--onto,main,main,feature",
				"GetCommitHash:HEAD",
				"CurrentBranch",
				"StashPop",
				"GetCommitHash:HEAD",
				"Push:feature",
			},
		},
		{
			name: "continue rebase",
			mock: func() *mockGit {
				m := newMockGit()
				m.isRepo = true
				m.isRebasing = true
				m.currentBranch = "feature"
				m.defaultBranch = "main"
				return m
			}(),
			cont: true,
			wantOps: []string{
				"IsRepo",
				"CurrentBranch",
				"DefaultBranch",
				"GetCommitHash:HEAD",
				"IsClean",
				"Stash:sage-sync-",
				"GetCommitHash:HEAD",
				"CurrentBranch",
				"Checkout:main",
				"PullFF",
				"Checkout:feature",
				"CurrentBranch",
				"Checkout:feature",
				"RunInteractive:rebase,--onto,main,main,feature",
				"GetCommitHash:HEAD",
				"CurrentBranch",
				"StashPop",
				"GetCommitHash:HEAD",
				"Push:feature",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SyncBranch(tt.mock, tt.abort, tt.cont)
			if (err != nil) != tt.wantErr {
				t.Errorf("SyncBranch() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				gotOps := tt.mock.operations
				for i, op := range gotOps {
					if strings.HasPrefix(op, "Stash:sage-sync-") {
						gotOps[i] = "Stash:sage-sync-"
					}
				}
				if !operationsMatch(gotOps, tt.wantOps) {
					t.Errorf("SyncBranch() operations = %v, want %v", gotOps, tt.wantOps)
				}
			}
		})
	}
}

// Helper function to compare operation slices
func operationsMatch(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func (m *mockGHClient) CreatePR(title, body, head, base string, draft bool) (*gh.PullRequest, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.pr, nil
}

func (m *mockGHClient) GetPRTemplate() (string, error) {
	return m.prTemplate, m.err
}

func (m *mockGHClient) AddLabels(prNum int, labels []string) error {
	return m.err
}

func (m *mockGHClient) RequestReviewers(prNum int, reviewers []string) error {
	return m.err
}

func (m *mockGHClient) ListPRs(state string) ([]gh.PullRequest, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.pr != nil {
		return []gh.PullRequest{*m.pr}, nil
	}
	return []gh.PullRequest{}, nil
}

func (m *mockGHClient) MergePR(prNum int, method string) error {
	return m.err
}

func (m *mockGHClient) ClosePR(prNum int) error {
	return m.err
}

func (m *mockGHClient) CheckoutPR(prNum int) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "pr-branch", nil
}

func (m *mockGHClient) GetPRDetails(prNum int) (*gh.PullRequest, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.pr, nil
}

func (m *mockGHClient) ListPRUnresolvedThreads(prNum int) ([]gh.UnresolvedThread, error) {
	return nil, m.err
}

func TestListPullRequests(t *testing.T) {
	tests := []struct {
		name    string
		ghc     *mockGHClient
		state   string
		wantNum int
		wantErr bool
	}{
		{
			name: "successful list",
			ghc: &mockGHClient{
				pr: &gh.PullRequest{
					Number: 123,
					Title:  "Test PR",
				},
			},
			state:   "open",
			wantNum: 1,
		},
		{
			name:  "empty list",
			ghc:   &mockGHClient{},
			state: "open",
		},
		{
			name: "list error",
			ghc: &mockGHClient{
				err: fmt.Errorf("list error"),
			},
			state:   "open",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prs, err := ListPRs(tt.ghc, tt.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListPRs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(prs) != tt.wantNum {
				t.Errorf("ListPRs() returned %v PRs, want %v", len(prs), tt.wantNum)
			}
		})
	}
}

func TestMergePullRequest(t *testing.T) {
	tests := []struct {
		name    string
		ghc     *mockGHClient
		prNum   int
		method  string
		wantErr bool
	}{
		{
			name:   "successful merge",
			ghc:    &mockGHClient{},
			prNum:  123,
			method: "squash",
		},
		{
			name: "merge error",
			ghc: &mockGHClient{
				err: fmt.Errorf("merge error"),
			},
			prNum:   123,
			method:  "squash",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MergePR(tt.ghc, tt.prNum, tt.method)
			if (err != nil) != tt.wantErr {
				t.Errorf("MergePR() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckoutPullRequest(t *testing.T) {
	tests := []struct {
		name       string
		mock       *mockGit
		ghc        *mockGHClient
		prNum      int
		wantBranch string
		wantErr    bool
	}{
		{
			name: "successful checkout",
			mock: func() *mockGit {
				m := newMockGit()
				m.isRepo = true
				return m
			}(),
			ghc:        &mockGHClient{},
			prNum:      123,
			wantBranch: "pr-branch",
		},
		{
			name: "checkout error",
			mock: func() *mockGit {
				m := newMockGit()
				m.isRepo = true
				return m
			}(),
			ghc: &mockGHClient{
				err: fmt.Errorf("checkout error"),
			},
			prNum:   123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branch, err := CheckoutPR(tt.mock, tt.ghc, tt.prNum)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckoutPR() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && branch != tt.wantBranch {
				t.Errorf("CheckoutPR() returned branch %v, want %v", branch, tt.wantBranch)
			}
		})
	}
}
