package app_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/crazywolf132/sage/internal/app"
)

// We'll define a minimal mock in this file for the GitService if needed
type mockGit struct {
	isRepo       bool
	isRepoErr    error
	isClean      bool
	isCleanErr   error
	commitErr    error
	stageErr     error
	pushErr      error
	curBranch    string
	curBranchErr error
}

func (m *mockGit) IsRepo() (bool, error) {
	if m.isRepoErr != nil {
		return false, m.isRepoErr
	}
	return m.isRepo, nil
}
func (m *mockGit) CurrentBranch() (string, error) {
	if m.curBranchErr != nil {
		return "", m.curBranchErr
	}
	if m.curBranch == "" {
		return "main", nil
	}
	return m.curBranch, nil
}
func (m *mockGit) IsClean() (bool, error) { return m.isClean, m.isCleanErr }
func (m *mockGit) StageAll() error        { return m.stageErr }
func (m *mockGit) Commit(msg string, allowEmpty bool) error {
	return m.commitErr
}
func (m *mockGit) Push(branch string, force bool) error {
	return m.pushErr
}

// The rest of the methods are not used in these tests, so no-ops:
func (m *mockGit) DefaultBranch() (string, error)               { return "main", nil }
func (m *mockGit) MergedBranches(base string) ([]string, error) { return nil, nil }
func (m *mockGit) DeleteBranch(name string) error               { return nil }
func (m *mockGit) FetchAll() error                              { return nil }
func (m *mockGit) Checkout(name string) error                   { return nil }
func (m *mockGit) Pull() error                                  { return nil }
func (m *mockGit) CreateBranch(name string) error               { return nil }
func (m *mockGit) Merge(base string) error                      { return nil }
func (m *mockGit) MergeAbort() error                            { return nil }
func (m *mockGit) IsMerging() (bool, error)                     { return false, nil }
func (m *mockGit) RebaseAbort() error                           { return nil }
func (m *mockGit) IsRebasing() (bool, error)                    { return false, nil }
func (m *mockGit) StatusPorcelain() (string, error)             { return "", nil }
func (m *mockGit) ResetSoft(ref string) error                   { return nil }
func (m *mockGit) ListBranches() ([]string, error)
func (m *mockGit) Log(branch string, limit int, stats, all bool) (string, error) { return "", nil }

func TestCommit_Success(t *testing.T) {
	g := &mockGit{
		isRepo:    true,
		isClean:   false, // changes present
		curBranch: "feature/test",
	}
	opts := app.CommitOptions{
		Message:         "Hello",
		UseConventional: true,
	}

	res, err := app.Commit(g, opts)
	assert.NoError(t, err)
	// Because we're using "UseConventional: true" and "Message: Hello",
	// it should become "chore: Hello" (per the code's logic).
	assert.Equal(t, "chore: Hello", res.ActualMessage)
	assert.False(t, res.Pushed)
}

func TestCommit_AllowEmpty(t *testing.T) {
	g := &mockGit{
		isRepo:  true,
		isClean: true, // no changes
	}
	opts := app.CommitOptions{
		Message:    "Empty Commit",
		AllowEmpty: true,
	}
	res, err := app.Commit(g, opts)
	assert.NoError(t, err)
	assert.Equal(t, "Empty Commit", res.ActualMessage)
}

func TestCommit_NoChangesAndNoEmpty(t *testing.T) {
	g := &mockGit{
		isRepo:  true,
		isClean: true, // no changes
	}
	opts := app.CommitOptions{Message: "test"}
	_, err := app.Commit(g, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no changes to commit")
}

func TestCommit_PushAfter(t *testing.T) {
	g := &mockGit{
		isRepo:    true,
		isClean:   false,
		curBranch: "feature/abc",
	}
	opts := app.CommitOptions{
		Message:         "PushMe",
		PushAfterCommit: true,
	}
	res, err := app.Commit(g, opts)
	assert.NoError(t, err)
	assert.Equal(t, "PushMe", res.ActualMessage)
	assert.True(t, res.Pushed)
}

func TestCommit_FailsIfNotRepo(t *testing.T) {
	g := &mockGit{
		isRepo: false,
	}
	_, err := app.Commit(g, app.CommitOptions{Message: "some msg"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repo")
}

func TestCommit_StageFails(t *testing.T) {
	g := &mockGit{
		isRepo:   true,
		isClean:  false,
		stageErr: errors.New("stage error"),
	}
	_, err := app.Commit(g, app.CommitOptions{Message: "some msg"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stage error")
}
