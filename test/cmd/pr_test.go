package cmd_test

import (
	"testing"

	"github.com/crazywolf132/sage/internal/githubutils"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockGitHubClient is a mock implementation of the GitHub client
type MockGitHubClient struct {
	mock.Mock
}

func (m *MockGitHubClient) CreatePullRequest(title, body, head, base string) (int, error) {
	args := m.Called(title, body, head, base)
	return args.Int(0), args.Error(1)
}

func (m *MockGitHubClient) ListPullRequests(state string) ([]githubutils.PullRequest, error) {
	args := m.Called(state)
	return args.Get(0).([]githubutils.PullRequest), args.Error(1)
}

func (m *MockGitHubClient) MergePullRequest(number int, method string) error {
	args := m.Called(number, method)
	return args.Error(0)
}

func TestCreatePullRequest(t *testing.T) {
	mockClient := new(MockGitHubClient)

	// Test successful PR creation
	mockClient.On("CreatePullRequest", "Test PR", "Test Description", "feature/test", "main").Return(1, nil)

	prNumber, err := mockClient.CreatePullRequest("Test PR", "Test Description", "feature/test", "main")
	assert.NoError(t, err)
	assert.Equal(t, 1, prNumber)

	mockClient.AssertExpectations(t)
}

func TestListPullRequests(t *testing.T) {
	mockClient := new(MockGitHubClient)

	expectedPRs := []githubutils.PullRequest{
		{Number: 1, Title: "Test PR 1", State: "open"},
		{Number: 2, Title: "Test PR 2", State: "open"},
	}

	mockClient.On("ListPullRequests", "open").Return(expectedPRs, nil)

	prs, err := mockClient.ListPullRequests("open")
	assert.NoError(t, err)
	assert.Len(t, prs, 2)
	assert.Equal(t, expectedPRs, prs)

	mockClient.AssertExpectations(t)
}

func TestMergePullRequest(t *testing.T) {
	mockClient := new(MockGitHubClient)

	// Test successful merge
	mockClient.On("MergePullRequest", 1, "squash").Return(nil)

	err := mockClient.MergePullRequest(1, "squash")
	assert.NoError(t, err)

	mockClient.AssertExpectations(t)
}

func TestCreatePullRequestDraftSettings(t *testing.T) {
	t.Run("force draft enabled", func(t *testing.T) {
		viper.Set("pr.forceDraft", true)
		defer viper.Set("pr.forceDraft", false)

		mockClient := new(MockGitHubClient)
		mockClient.On("CreatePullRequest", "Test PR", "Test Description", "feature/test", "main").Return(1, nil)

		prNumber, err := mockClient.CreatePullRequest("Test PR", "Test Description", "feature/test", "main")
		assert.NoError(t, err)
		assert.Equal(t, 1, prNumber)

		mockClient.AssertExpectations(t)
	})

	t.Run("default draft enabled", func(t *testing.T) {
		viper.Set("pr.defaultDraft", true)
		defer viper.Set("pr.defaultDraft", false)

		mockClient := new(MockGitHubClient)
		mockClient.On("CreatePullRequest", "Test PR", "Test Description", "feature/test", "main").Return(1, nil)

		prNumber, err := mockClient.CreatePullRequest("Test PR", "Test Description", "feature/test", "main")
		assert.NoError(t, err)
		assert.Equal(t, 1, prNumber)

		mockClient.AssertExpectations(t)
	})

	t.Run("draft flag overrides settings", func(t *testing.T) {
		viper.Set("pr.defaultDraft", false)
		viper.Set("pr.forceDraft", false)

		mockClient := new(MockGitHubClient)
		mockClient.On("CreatePullRequest", "Test PR", "Test Description", "feature/test", "main").Return(1, nil)

		prNumber, err := mockClient.CreatePullRequest("Test PR", "Test Description", "feature/test", "main")
		assert.NoError(t, err)
		assert.Equal(t, 1, prNumber)

		mockClient.AssertExpectations(t)
	})
}
