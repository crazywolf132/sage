package app_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/crazywolf132/sage/internal/app"
)

type mockGitClean struct {
	isRepo     bool
	isRepoErr  error
	defBranch  string
	defBranchE error
	curBranch  string
	curBranchE error
	merged     []string
	mergedErr  error
}

func (m *mockGitClean) IsRepo() (bool, error) { return m.isRepo, m.isRepoErr }
func (m *mockGitClean) DefaultBranch() (string, error) {
	if m.defBranch == "" {
		return "main", m.defBranchE
	}
	return m.defBranch, m.defBranchE
}
func (m *mockGitClean) CurrentBranch() (string, error) {
	if m.curBranch == "" {
		return "feature/xyz", m.curBranchE
	}
	return m.curBranch, m.curBranchE
}
func (m *mockGitClean) MergedBranches(base string) ([]string, error) {
	return m.merged, m.mergedErr
}

// partial implement:
func (m *mockGitClean) IsClean() (bool, error)              { return false, nil }
func (m *mockGitClean) StageAll() error                     { return nil }
func (m *mockGitClean) Commit(msg string, empty bool) error { return nil }
func (m *mockGitClean) Push(b string, f bool) error         { return nil }
func (m *mockGitClean) DeleteBranch(name string) error      { return nil }
func (m *mockGitClean) FetchAll() error                     { return nil }
func (m *mockGitClean) Checkout(name string) error          { return nil }
func (m *mockGitClean) Pull() error                         { return nil }
func (m *mockGitClean) CreateBranch(name string) error      { return nil }
func (m *mockGitClean) Merge(base string) error             { return nil }
func (m *mockGitClean) MergeAbort() error                   { return nil }
func (m *mockGitClean) IsMerging() (bool, error)            { return false, nil }
func (m *mockGitClean) RebaseAbort() error                  { return nil }
func (m *mockGitClean) IsRebasing() (bool, error)           { return false, nil }
func (m *mockGitClean) StatusPorcelain() (string, error)    { return "", nil }
func (m *mockGitClean) ResetSoft(ref string) error          { return nil }
func (m *mockGitClean) ListBranches() ([]string, error)
func (m *mockGitClean) Log(branch string, limit int, stats, all bool) (string, error) {
	return "", nil
}

func TestFindCleanableBranches(t *testing.T) {
	g := &mockGitClean{
		isRepo:    true,
		defBranch: "main",
		curBranch: "feature/abc",
		merged: []string{
			"main", "feature/abc", "feature/123", "bugfix/wip",
			"", "  ",
		},
	}

	info, err := app.FindCleanableBranches(g)
	assert.NoError(t, err)
	assert.Equal(t, []string{"feature/123", "bugfix/wip"}, info.Branches)
}

func TestFindCleanableBranches_NotRepo(t *testing.T) {
	g := &mockGitClean{isRepo: false}
	_, err := app.FindCleanableBranches(g)
	assert.Error(t, err)
}

func TestDeleteLocalBranches(t *testing.T) {
	mockDel := &delGit{
		deletes: map[string]error{
			"feature/123": nil,
			"bugfix/wip":  errors.New("not fully merged"),
		},
	}
	results := app.DeleteLocalBranches(mockDel, []string{"feature/123", "bugfix/wip", "random"})
	assert.Len(t, results, 3)
	assert.Nil(t, results[0].Err)
	assert.NotNil(t, results[1].Err)
	assert.Nil(t, results[2].Err) // "random" not in map => no error
}

type delGit struct{ deletes map[string]error }

func (d *delGit) DeleteBranch(name string) error {
	if e, ok := d.deletes[name]; ok {
		return e
	}
	return nil
}

// no-ops for the rest
func (d *delGit) IsRepo() (bool, error)                                         { return true, nil }
func (d *delGit) CurrentBranch() (string, error)                                { return "main", nil }
func (d *delGit) IsClean() (bool, error)                                        { return false, nil }
func (d *delGit) StageAll() error                                               { return nil }
func (d *delGit) Commit(string, bool) error                                     { return nil }
func (d *delGit) Push(string, bool) error                                       { return nil }
func (d *delGit) DefaultBranch() (string, error)                                { return "main", nil }
func (d *delGit) MergedBranches(string) ([]string, error)                       { return nil, nil }
func (d *delGit) FetchAll() error                                               { return nil }
func (d *delGit) Checkout(string) error                                         { return nil }
func (d *delGit) Pull() error                                                   { return nil }
func (d *delGit) CreateBranch(string) error                                     { return nil }
func (d *delGit) Merge(string) error                                            { return nil }
func (d *delGit) MergeAbort() error                                             { return nil }
func (d *delGit) IsMerging() (bool, error)                                      { return false, nil }
func (d *delGit) RebaseAbort() error                                            { return nil }
func (d *delGit) IsRebasing() (bool, error)                                     { return false, nil }
func (d *delGit) StatusPorcelain() (string, error)                              { return "", nil }
func (d *delGit) ResetSoft(ref string) error                                    { return nil }
func (d *delGit) ListBranches() ([]string, error)                               { return nil, nil }
func (m *delGit) Log(branch string, limit int, stats, all bool) (string, error) { return "", nil }
