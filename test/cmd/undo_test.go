package cmd_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/crazywolf132/sage/cmd"
	"github.com/crazywolf132/sage/internal/gitutils"
)

func TestUndoCommand(t *testing.T) {
	t.Run("successful undo", func(t *testing.T) {
		// Create a fresh mock for this test
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Setup mock expectations
		mockGit.On("RunGitCommand", []string{"rev-parse", "--git-dir"}).Return(nil).Once()
		mockGit.On("IsMergeInProgress").Return(false, nil).Once()
		mockGit.On("IsRebaseInProgress").Return(false, nil).Once()
		mockGit.On("RunGitCommand", []string{"reset", "--soft", "HEAD~1"}).Return(nil).Once()

		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"undo"})
		err := rootCmd.Execute()

		// Assert
		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("undo during merge", func(t *testing.T) {
		// Create a fresh mock for this test
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Setup mock expectations
		mockGit.On("RunGitCommand", []string{"rev-parse", "--git-dir"}).Return(nil).Once()
		mockGit.On("IsMergeInProgress").Return(true, nil).Once()
		mockGit.On("RunGitCommand", []string{"merge", "--abort"}).Return(nil).Once()

		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"undo"})
		err := rootCmd.Execute()

		// Assert
		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("undo during rebase", func(t *testing.T) {
		// Create a fresh mock for this test
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Setup mock expectations
		mockGit.On("RunGitCommand", []string{"rev-parse", "--git-dir"}).Return(nil).Once()
		mockGit.On("IsMergeInProgress").Return(false, nil).Once()
		mockGit.On("IsRebaseInProgress").Return(true, nil).Once()
		mockGit.On("RunGitCommand", []string{"rebase", "--abort"}).Return(nil).Once()

		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"undo"})
		err := rootCmd.Execute()

		// Assert
		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("undo with no commits", func(t *testing.T) {
		// Create a fresh mock for this test
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Setup mock expectations
		mockGit.On("RunGitCommand", []string{"rev-parse", "--git-dir"}).Return(nil).Once()
		mockGit.On("IsMergeInProgress").Return(false, nil).Once()
		mockGit.On("IsRebaseInProgress").Return(false, nil).Once()
		mockGit.On("RunGitCommand", []string{"reset", "--soft", "HEAD~1"}).Return(fmt.Errorf("fatal: ambiguous argument 'HEAD~1': unknown revision")).Once()

		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"undo"})
		err := rootCmd.Execute()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown revision")
		mockGit.AssertExpectations(t)
	})

	t.Run("undo with arguments", func(t *testing.T) {
		// Create a fresh mock for this test
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"undo", "extra"})
		err := rootCmd.Execute()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown command \"extra\" for \"sage undo\"")
		mockGit.AssertExpectations(t)
	})

	t.Run("not a git repository", func(t *testing.T) {
		// Create a fresh mock for this test
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Setup mock expectations
		mockGit.On("RunGitCommand", []string{"rev-parse", "--git-dir"}).Return(fmt.Errorf("fatal: not a git repository")).Once()

		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"undo"})
		err := rootCmd.Execute()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a git repository")
		mockGit.AssertExpectations(t)
	})

	t.Run("merge check fails", func(t *testing.T) {
		// Create a fresh mock for this test
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Setup mock expectations
		mockGit.On("RunGitCommand", []string{"rev-parse", "--git-dir"}).Return(nil).Once()
		mockGit.On("IsMergeInProgress").Return(false, fmt.Errorf("failed to check merge status")).Once()

		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"undo"})
		err := rootCmd.Execute()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check merge status")
		mockGit.AssertExpectations(t)
	})

	t.Run("rebase check fails", func(t *testing.T) {
		// Create a fresh mock for this test
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Setup mock expectations
		mockGit.On("RunGitCommand", []string{"rev-parse", "--git-dir"}).Return(nil).Once()
		mockGit.On("IsMergeInProgress").Return(false, nil).Once()
		mockGit.On("IsRebaseInProgress").Return(false, fmt.Errorf("failed to check rebase status")).Once()

		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"undo"})
		err := rootCmd.Execute()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check rebase status")
		mockGit.AssertExpectations(t)
	})

	t.Run("merge abort fails", func(t *testing.T) {
		// Create a fresh mock for this test
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Setup mock expectations
		mockGit.On("RunGitCommand", []string{"rev-parse", "--git-dir"}).Return(nil).Once()
		mockGit.On("IsMergeInProgress").Return(true, nil).Once()
		mockGit.On("RunGitCommand", []string{"merge", "--abort"}).Return(fmt.Errorf("failed to abort merge")).Once()

		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"undo"})
		err := rootCmd.Execute()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to abort merge")
		mockGit.AssertExpectations(t)
	})

	t.Run("rebase abort fails", func(t *testing.T) {
		// Create a fresh mock for this test
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Setup mock expectations
		mockGit.On("RunGitCommand", []string{"rev-parse", "--git-dir"}).Return(nil).Once()
		mockGit.On("IsMergeInProgress").Return(false, nil).Once()
		mockGit.On("IsRebaseInProgress").Return(true, nil).Once()
		mockGit.On("RunGitCommand", []string{"rebase", "--abort"}).Return(fmt.Errorf("failed to abort rebase")).Once()

		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"undo"})
		err := rootCmd.Execute()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to abort rebase")
		mockGit.AssertExpectations(t)
	})
}
