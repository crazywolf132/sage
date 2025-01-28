package gitutils_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/crazywolf132/sage/internal/gitutils"
)

// Additional test cases for merge and rebase scenarios can be added here

func TestMergeInProgress(t *testing.T) {
	mock := new(gitutils.MockGitRunner)

	// Test when merge is in progress
	mock.On("IsMergeInProgress").Return(true, nil).Once()
	mergeInProgress, err := mock.IsMergeInProgress()
	assert.NoError(t, err)
	assert.True(t, mergeInProgress)

	// Test when no merge is in progress
	mock.On("IsMergeInProgress").Return(false, nil).Once()
	mergeInProgress, err = mock.IsMergeInProgress()
	assert.NoError(t, err)
	assert.False(t, mergeInProgress)

	// Test error case
	expectedErr := fmt.Errorf("failed to check merge status")
	mock.On("IsMergeInProgress").Return(false, expectedErr).Once()
	mergeInProgress, err = mock.IsMergeInProgress()
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.False(t, mergeInProgress)

	mock.AssertExpectations(t)
}

func TestRebaseInProgress(t *testing.T) {
	mock := new(gitutils.MockGitRunner)

	// Test when rebase is in progress
	mock.On("IsRebaseInProgress").Return(true, nil).Once()
	rebaseInProgress, err := mock.IsRebaseInProgress()
	assert.NoError(t, err)
	assert.True(t, rebaseInProgress)

	// Test when no rebase is in progress
	mock.On("IsRebaseInProgress").Return(false, nil).Once()
	rebaseInProgress, err = mock.IsRebaseInProgress()
	assert.NoError(t, err)
	assert.False(t, rebaseInProgress)

	// Test error case
	expectedErr := fmt.Errorf("failed to check rebase status")
	mock.On("IsRebaseInProgress").Return(false, expectedErr).Once()
	rebaseInProgress, err = mock.IsRebaseInProgress()
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.False(t, rebaseInProgress)

	mock.AssertExpectations(t)
}
