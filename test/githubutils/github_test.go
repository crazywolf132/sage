package githubutils_test

import (
	"testing"

	"github.com/crazywolf132/sage/internal/githubutils"
	"github.com/stretchr/testify/assert"
)

func TestCreatePRParams(t *testing.T) {
	params := githubutils.CreatePRParams{
		Title: "Test PR",
		Head:  "feature-branch",
		Base:  "main",
		Body:  "Test PR description",
		Draft: true,
	}

	assert.Equal(t, "Test PR", params.Title)
	assert.Equal(t, "feature-branch", params.Head)
	assert.Equal(t, "main", params.Base)
	assert.Equal(t, "Test PR description", params.Body)
	assert.True(t, params.Draft)
}

func TestPullRequest(t *testing.T) {
	pr := githubutils.PullRequest{
		Number:  123,
		HTMLURL: "https://github.com/owner/repo/pull/123",
		Title:   "Test PR",
		State:   "open",
		Body:    "Test PR description",
	}

	assert.Equal(t, 123, pr.Number)
	assert.Equal(t, "https://github.com/owner/repo/pull/123", pr.HTMLURL)
	assert.Equal(t, "Test PR", pr.Title)
	assert.Equal(t, "open", pr.State)
	assert.Equal(t, "Test PR description", pr.Body)
}
