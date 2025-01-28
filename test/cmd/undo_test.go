package cmd_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/crazywolf132/sage/cmd"
	"github.com/crazywolf132/sage/internal/gitutils"
)

func TestUndoCommand(t *testing.T) {
	t.Run("successful_undo", func(t *testing.T) {
		// Create a fresh mock for this test
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// We expect 4 calls here, in this order:
		// 1. RunGitCommand("rev-parse", "--git-dir")   -> checks if it's a Git repo
		// 2. IsMergeInProgress()
		// 3. IsRebaseInProgress()
		// 4. RunGitCommand("reset", "--soft", "HEAD~1")
		mockGit.On("RunGitCommand", "rev-parse", "--git-dir").Return(nil).Once()
		mockGit.On("IsMergeInProgress").Return(false, nil).Once()
		mockGit.On("IsRebaseInProgress").Return(false, nil).Once()
		mockGit.On("RunGitCommand", "reset", "--soft", "HEAD~1").Return(nil).Once()

		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"undo"})
		err := rootCmd.Execute()

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("undo_during_merge", func(t *testing.T) {
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Calls:
		// 1. RunGitCommand("rev-parse", "--git-dir")
		// 2. IsMergeInProgress() -> returns true
		// 3. RunGitCommand("merge", "--abort")
		mockGit.On("RunGitCommand", "rev-parse", "--git-dir").Return(nil).Once()
		mockGit.On("IsMergeInProgress").Return(true, nil).Once()
		// Because inMerge==true, we skip rebase check & skip the reset call
		mockGit.On("RunGitCommand", "merge", "--abort").Return(nil).Once()

		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"undo"})
		err := rootCmd.Execute()

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("undo_during_rebase", func(t *testing.T) {
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Calls:
		// 1. RunGitCommand("rev-parse", "--git-dir")
		// 2. IsMergeInProgress() -> returns false
		// 3. IsRebaseInProgress() -> returns true
		// 4. RunGitCommand("rebase", "--abort")
		mockGit.On("RunGitCommand", "rev-parse", "--git-dir").Return(nil).Once()
		mockGit.On("IsMergeInProgress").Return(false, nil).Once()
		mockGit.On("IsRebaseInProgress").Return(true, nil).Once()
		mockGit.On("RunGitCommand", "rebase", "--abort").Return(nil).Once()

		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"undo"})
		err := rootCmd.Execute()

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("undo_with_no_commits", func(t *testing.T) {
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Calls:
		// 1. RunGitCommand("rev-parse", "--git-dir")
		// 2. IsMergeInProgress() -> false
		// 3. IsRebaseInProgress() -> false
		// 4. RunGitCommand("reset", "--soft", "HEAD~1") -> but HEAD~1 doesn’t exist
		mockGit.On("RunGitCommand", "rev-parse", "--git-dir").Return(nil).Once()
		mockGit.On("IsMergeInProgress").Return(false, nil).Once()
		mockGit.On("IsRebaseInProgress").Return(false, nil).Once()
		mockGit.
			On("RunGitCommand", "reset", "--soft", "HEAD~1").
			Return(fmt.Errorf("fatal: ambiguous argument 'HEAD~1': unknown revision")).
			Once()

		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"undo"})
		err := rootCmd.Execute()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown revision")
		mockGit.AssertExpectations(t)
	})

	t.Run("undo_with_arguments", func(t *testing.T) {
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// If user tries: `sage undo extra`
		// This will fail because "undo" doesn't accept extra positional args
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"undo", "extra"})
		err := rootCmd.Execute()

		assert.Error(t, err)
		// Typically you’d see an “unknown command 'extra' for 'sage undo'”
		assert.Contains(t, err.Error(), "unknown command \"extra\" for \"sage undo\"")
		mockGit.AssertExpectations(t)
	})

	t.Run("not_a_git_repository", func(t *testing.T) {
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		mockGit.On("RunGitCommand", "rev-parse", "--git-dir").
			Return(fmt.Errorf("fatal: not a git repository")).Once()

		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"undo"})
		err := rootCmd.Execute()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a git repository")
		mockGit.AssertExpectations(t)
	})

	t.Run("merge_check_fails", func(t *testing.T) {
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		mockGit.On("RunGitCommand", "rev-parse", "--git-dir").Return(nil).Once()
		mockGit.On("IsMergeInProgress").Return(false, fmt.Errorf("failed to check merge status")).Once()

		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"undo"})
		err := rootCmd.Execute()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check merge status")
		mockGit.AssertExpectations(t)
	})

	t.Run("rebase_check_fails", func(t *testing.T) {
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		mockGit.On("RunGitCommand", "rev-parse", "--git-dir").Return(nil).Once()
		mockGit.On("IsMergeInProgress").Return(false, nil).Once()
		mockGit.On("IsRebaseInProgress").Return(false, fmt.Errorf("failed to check rebase status")).Once()

		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"undo"})
		err := rootCmd.Execute()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check rebase status")
		mockGit.AssertExpectations(t)
	})

	t.Run("merge_abort_fails", func(t *testing.T) {
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		mockGit.On("RunGitCommand", "rev-parse", "--git-dir").Return(nil).Once()
		mockGit.On("IsMergeInProgress").Return(true, nil).Once()
		mockGit.On("RunGitCommand", "merge", "--abort").Return(fmt.Errorf("failed to abort merge")).Once()

		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"undo"})
		err := rootCmd.Execute()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to abort merge")
		mockGit.AssertExpectations(t)
	})

	t.Run("rebase_abort_fails", func(t *testing.T) {
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		mockGit.On("RunGitCommand", "rev-parse", "--git-dir").Return(nil).Once()
		mockGit.On("IsMergeInProgress").Return(false, nil).Once()
		mockGit.On("IsRebaseInProgress").Return(true, nil).Once()
		mockGit.On("RunGitCommand", "rebase", "--abort").Return(fmt.Errorf("failed to abort rebase")).Once()

		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"undo"})
		err := rootCmd.Execute()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to abort rebase")
		mockGit.AssertExpectations(t)
	})
}
