package gh

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHTTPClient implements http.RoundTripper interface
type mockHTTPClient struct {
	responses map[string]struct {
		statusCode int
		body       string
	}
}

func (m *mockHTTPClient) RoundTrip(req *http.Request) (*http.Response, error) {
	key := fmt.Sprintf("%s %s", req.Method, req.URL.Path)
	if req.URL.RawQuery != "" {
		key = fmt.Sprintf("%s?%s", key, req.URL.RawQuery)
	}
	if resp, ok := m.responses[key]; ok {
		return &http.Response{
			StatusCode: resp.statusCode,
			Body:       io.NopCloser(strings.NewReader(resp.body)),
			Header:     make(http.Header),
		}, nil
	}
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader(`{"message": "Not Found"}`)),
		Header:     make(http.Header),
	}, nil
}

func TestCreatePR(t *testing.T) {
	mock := &mockHTTPClient{
		responses: map[string]struct {
			statusCode int
			body       string
		}{
			"POST /repos/owner/repo/pulls": {
				statusCode: http.StatusCreated,
				body: `{
					"number": 123,
					"title": "Test PR",
					"body": "PR description",
					"state": "open",
					"html_url": "https://github.com/owner/repo/pull/123",
					"draft": false,
					"merged": false,
					"head": {"ref": "feature-branch"},
					"base": {"ref": "main"}
				}`,
			},
		},
	}

	client := &pullRequestAPI{
		owner:  "owner",
		repo:   "repo",
		token:  "test-token",
		client: &http.Client{Transport: mock},
	}

	pr, err := client.CreatePR("Test PR", "PR description", "feature-branch", "main", false)
	require.NoError(t, err)
	assert.Equal(t, 123, pr.Number)
	assert.Equal(t, "Test PR", pr.Title)
	assert.Equal(t, "feature-branch", pr.Head.Ref)
	assert.Equal(t, "main", pr.Base.Ref)
}

func TestListPRs(t *testing.T) {
	mock := &mockHTTPClient{
		responses: map[string]struct {
			statusCode int
			body       string
		}{
			"GET /repos/owner/repo/pulls?state=open": {
				statusCode: http.StatusOK,
				body: `[
					{
						"number": 123,
						"title": "First PR",
						"state": "open",
						"head": {"ref": "feature-1"},
						"base": {"ref": "main"}
					},
					{
						"number": 124,
						"title": "Second PR",
						"state": "open",
						"head": {"ref": "feature-2"},
						"base": {"ref": "main"}
					}
				]`,
			},
		},
	}

	client := &pullRequestAPI{
		owner:  "owner",
		repo:   "repo",
		token:  "test-token",
		client: &http.Client{Transport: mock},
	}

	prs, err := client.ListPRs("open")
	require.NoError(t, err)
	assert.Len(t, prs, 2)
	assert.Equal(t, 123, prs[0].Number)
	assert.Equal(t, "First PR", prs[0].Title)
	assert.Equal(t, "feature-1", prs[0].Head.Ref)
}

func TestMergePR(t *testing.T) {
	mock := &mockHTTPClient{
		responses: map[string]struct {
			statusCode int
			body       string
		}{
			"GET /repos/owner/repo": {
				statusCode: http.StatusOK,
				body: `{
					"allow_merge_commit": true,
					"allow_squash_merge": true,
					"allow_rebase_merge": true
				}`,
			},
			"PUT /repos/owner/repo/pulls/123/merge": {
				statusCode: http.StatusOK,
				body:       `{"message": "Pull Request successfully merged"}`,
			},
		},
	}

	client := &pullRequestAPI{
		owner:  "owner",
		repo:   "repo",
		token:  "test-token",
		client: &http.Client{Transport: mock},
	}

	err := client.MergePR(123, "merge")
	require.NoError(t, err)
}

func TestGetPRDetails(t *testing.T) {
	mock := &mockHTTPClient{
		responses: map[string]struct {
			statusCode int
			body       string
		}{
			"GET /repos/owner/repo/pulls/123": {
				statusCode: http.StatusOK,
				body: `{
					"number": 123,
					"title": "Test PR",
					"body": "PR description",
					"state": "open",
					"html_url": "https://github.com/owner/repo/pull/123",
					"draft": false,
					"merged": false,
					"head": {"ref": "feature-branch"},
					"base": {"ref": "main"}
				}`,
			},
		},
	}

	client := &pullRequestAPI{
		owner:  "owner",
		repo:   "repo",
		token:  "test-token",
		client: &http.Client{Transport: mock},
	}

	pr, err := client.GetPRDetails(123)
	require.NoError(t, err)
	assert.Equal(t, 123, pr.Number)
	assert.Equal(t, "Test PR", pr.Title)
	assert.Equal(t, "feature-branch", pr.Head.Ref)
	assert.Equal(t, "main", pr.Base.Ref)
}

func TestListPRUnresolvedThreads(t *testing.T) {
	mock := &mockHTTPClient{
		responses: map[string]struct {
			statusCode int
			body       string
		}{
			"GET /repos/owner/repo/pulls/123/comments": {
				statusCode: http.StatusOK,
				body: `[
					{
						"id": 1,
						"body": "Need to fix this",
						"path": "file.go",
						"line": 10,
						"created_at": "2023-01-01T00:00:00Z",
						"user": {"login": "reviewer"},
						"diff_hunk": "@@ -10,6 +10,6 @@ func example() {}"
					}
				]`,
			},
		},
	}

	client := &pullRequestAPI{
		owner:  "owner",
		repo:   "repo",
		token:  "test-token",
		client: &http.Client{Transport: mock},
	}

	threads, err := client.ListPRUnresolvedThreads(123)
	require.NoError(t, err)
	assert.Len(t, threads, 1)
	assert.Equal(t, "file.go", threads[0].Path)
	assert.Equal(t, 10, threads[0].Line)
	assert.Equal(t, "reviewer", threads[0].Comments[0].User)
	assert.Equal(t, "Need to fix this", threads[0].Comments[0].Body)
}

func TestGetLatestRelease(t *testing.T) {
	mock := &mockHTTPClient{
		responses: map[string]struct {
			statusCode int
			body       string
		}{
			"GET /repos/owner/repo/releases/latest": {
				statusCode: http.StatusOK,
				body: `{
					"tag_name": "v1.2.3"
				}`,
			},
		},
	}

	client := &pullRequestAPI{
		owner:  "owner",
		repo:   "repo",
		token:  "test-token",
		client: &http.Client{Transport: mock},
	}

	version, err := client.GetLatestRelease()
	require.NoError(t, err)
	assert.Equal(t, "1.2.3", version)
}

func TestNewClient(t *testing.T) {
	// Save original environment
	originalOwner := os.Getenv("SAGE_GITHUB_OWNER")
	originalRepo := os.Getenv("SAGE_GITHUB_REPO")
	originalToken := os.Getenv("SAGE_GITHUB_TOKEN")
	defer func() {
		os.Setenv("SAGE_GITHUB_OWNER", originalOwner)
		os.Setenv("SAGE_GITHUB_REPO", originalRepo)
		os.Setenv("SAGE_GITHUB_TOKEN", originalToken)
	}()

	// Set test environment variables
	os.Setenv("SAGE_GITHUB_OWNER", "test-owner")
	os.Setenv("SAGE_GITHUB_REPO", "test-repo")
	os.Setenv("SAGE_GITHUB_TOKEN", "test-token")

	client := NewClient()
	prAPI, ok := client.(*pullRequestAPI)
	require.True(t, ok)
	assert.Equal(t, "test-owner", prAPI.owner)
	assert.Equal(t, "test-repo", prAPI.repo)
	assert.Equal(t, "test-token", prAPI.token)
}

func TestGetToken(t *testing.T) {
	// Save original environment
	originalSageToken := os.Getenv("SAGE_GITHUB_TOKEN")
	originalGithubToken := os.Getenv("GITHUB_TOKEN")
	defer func() {
		os.Setenv("SAGE_GITHUB_TOKEN", originalSageToken)
		os.Setenv("GITHUB_TOKEN", originalGithubToken)
	}()

	tests := []struct {
		name           string
		sageToken      string
		githubToken    string
		expectedToken  string
		expectedSource string
	}{
		{
			name:           "SAGE_GITHUB_TOKEN takes precedence",
			sageToken:      "sage-token",
			githubToken:    "github-token",
			expectedToken:  "sage-token",
			expectedSource: "SAGE_GITHUB_TOKEN environment variable",
		},
		{
			name:           "Falls back to GITHUB_TOKEN",
			sageToken:      "",
			githubToken:    "github-token",
			expectedToken:  "github-token",
			expectedSource: "GITHUB_TOKEN environment variable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("SAGE_GITHUB_TOKEN", tt.sageToken)
			os.Setenv("GITHUB_TOKEN", tt.githubToken)

			tokenSource := getToken()
			assert.Equal(t, tt.expectedToken, tokenSource.Token)
			assert.Equal(t, tt.expectedSource, tokenSource.Source)
		})
	}
}
