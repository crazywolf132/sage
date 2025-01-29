package githubutils_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/crazywolf132/sage/internal/githubutils"
)

func TestCreatePullRequest(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/repos/owner/repo/pulls", r.URL.Path)
			assert.Equal(t, "token test-token", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var params githubutils.CreatePRParams
			err := json.NewDecoder(r.Body).Decode(&params)
			require.NoError(t, err)

			assert.Equal(t, "Test PR", params.Title)
			assert.Equal(t, "feature", params.Head)
			assert.Equal(t, "main", params.Base)
			assert.Equal(t, "Test description", params.Body)
			assert.True(t, params.Draft)

			response := githubutils.PullRequest{
				Number:  123,
				HTMLURL: "https://github.com/owner/repo/pull/123",
				Title:   params.Title,
				State:   "open",
				Body:    params.Body,
			}
			err = json.NewEncoder(w).Encode(response)
			require.NoError(t, err)
		}))
		defer server.Close()

		oldClient := githubutils.DefaultClient
		oldBaseURL := githubutils.BaseURL
		githubutils.DefaultClient = server.Client()
		githubutils.BaseURL = server.URL
		defer func() {
			githubutils.DefaultClient = oldClient
			githubutils.BaseURL = oldBaseURL
		}()

		params := githubutils.CreatePRParams{
			Title: "Test PR",
			Head:  "feature",
			Base:  "main",
			Body:  "Test description",
			Draft: true,
		}

		pr, err := githubutils.CreatePullRequest("test-token", "owner", "repo", params)
		assert.NoError(t, err)
		assert.Equal(t, 123, pr.Number)
		assert.Equal(t, "https://github.com/owner/repo/pull/123", pr.HTMLURL)
		assert.Equal(t, "Test PR", pr.Title)
		assert.Equal(t, "open", pr.State)
		assert.Equal(t, "Test description", pr.Body)
	})

	t.Run("no token", func(t *testing.T) {
		_, err := githubutils.CreatePullRequest("", "owner", "repo", githubutils.CreatePRParams{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no GitHub token provided")
	})

	t.Run("api error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnprocessableEntity)
		}))
		defer server.Close()

		oldClient := githubutils.DefaultClient
		oldBaseURL := githubutils.BaseURL
		githubutils.DefaultClient = server.Client()
		githubutils.BaseURL = server.URL
		defer func() {
			githubutils.DefaultClient = oldClient
			githubutils.BaseURL = oldBaseURL
		}()

		_, err := githubutils.CreatePullRequest("test-token", "owner", "repo", githubutils.CreatePRParams{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "GitHub API returned status 422")
	})
}

func TestListPullRequests(t *testing.T) {
	t.Run("successful list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/repos/owner/repo/pulls", r.URL.Path)
			assert.Equal(t, "open", r.URL.Query().Get("state"))
			assert.Equal(t, "token test-token", r.Header.Get("Authorization"))

			prs := []githubutils.PullRequest{
				{Number: 1, Title: "PR 1", State: "open"},
				{Number: 2, Title: "PR 2", State: "open"},
			}
			err := json.NewEncoder(w).Encode(prs)
			require.NoError(t, err)
		}))
		defer server.Close()

		oldClient := githubutils.DefaultClient
		oldBaseURL := githubutils.BaseURL
		githubutils.DefaultClient = server.Client()
		githubutils.BaseURL = server.URL
		defer func() {
			githubutils.DefaultClient = oldClient
			githubutils.BaseURL = oldBaseURL
		}()

		prs, err := githubutils.ListPullRequests("test-token", "owner", "repo", "open")
		assert.NoError(t, err)
		assert.Len(t, prs, 2)
		assert.Equal(t, "PR 1", prs[0].Title)
		assert.Equal(t, "PR 2", prs[1].Title)
	})

	t.Run("no token", func(t *testing.T) {
		_, err := githubutils.ListPullRequests("", "owner", "repo", "open")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no GitHub token provided")
	})

	t.Run("api error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		oldClient := githubutils.DefaultClient
		oldBaseURL := githubutils.BaseURL
		githubutils.DefaultClient = server.Client()
		githubutils.BaseURL = server.URL
		defer func() {
			githubutils.DefaultClient = oldClient
			githubutils.BaseURL = oldBaseURL
		}()

		_, err := githubutils.ListPullRequests("test-token", "owner", "repo", "open")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "GitHub API returned status 401")
	})
}

func TestGetPullRequest(t *testing.T) {
	t.Run("successful get", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/repos/owner/repo/pulls/123", r.URL.Path)
			assert.Equal(t, "token test-token", r.Header.Get("Authorization"))

			pr := githubutils.PullRequest{
				Number:  123,
				Title:   "Test PR",
				State:   "open",
				HTMLURL: "https://github.com/owner/repo/pull/123",
			}
			err := json.NewEncoder(w).Encode(pr)
			require.NoError(t, err)
		}))
		defer server.Close()

		oldClient := githubutils.DefaultClient
		oldBaseURL := githubutils.BaseURL
		githubutils.DefaultClient = server.Client()
		githubutils.BaseURL = server.URL
		defer func() {
			githubutils.DefaultClient = oldClient
			githubutils.BaseURL = oldBaseURL
		}()

		pr, err := githubutils.GetPullRequest("test-token", "owner", "repo", 123)
		assert.NoError(t, err)
		assert.Equal(t, 123, pr.Number)
		assert.Equal(t, "Test PR", pr.Title)
		assert.Equal(t, "open", pr.State)
	})

	t.Run("api error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		oldClient := githubutils.DefaultClient
		oldBaseURL := githubutils.BaseURL
		githubutils.DefaultClient = server.Client()
		githubutils.BaseURL = server.URL
		defer func() {
			githubutils.DefaultClient = oldClient
			githubutils.BaseURL = oldBaseURL
		}()

		_, err := githubutils.GetPullRequest("test-token", "owner", "repo", 123)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "GitHub API returned status 404")
	})
}

func TestClosePullRequest(t *testing.T) {
	t.Run("successful close", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "PATCH", r.Method)
			assert.Equal(t, "/repos/owner/repo/pulls/123", r.URL.Path)
			assert.Equal(t, "token test-token", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var body map[string]string
			err := json.NewDecoder(r.Body).Decode(&body)
			require.NoError(t, err)
			assert.Equal(t, "closed", body["state"])

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		oldClient := githubutils.DefaultClient
		oldBaseURL := githubutils.BaseURL
		githubutils.DefaultClient = server.Client()
		githubutils.BaseURL = server.URL
		defer func() {
			githubutils.DefaultClient = oldClient
			githubutils.BaseURL = oldBaseURL
		}()

		err := githubutils.ClosePullRequest("test-token", "owner", "repo", 123)
		assert.NoError(t, err)
	})

	t.Run("api error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		oldClient := githubutils.DefaultClient
		oldBaseURL := githubutils.BaseURL
		githubutils.DefaultClient = server.Client()
		githubutils.BaseURL = server.URL
		defer func() {
			githubutils.DefaultClient = oldClient
			githubutils.BaseURL = oldBaseURL
		}()

		err := githubutils.ClosePullRequest("test-token", "owner", "repo", 123)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "GitHub API returned status 401")
	})
}

func TestMergePullRequest(t *testing.T) {
	t.Run("successful merge", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "PUT", r.Method)
			assert.Equal(t, "/repos/owner/repo/pulls/123/merge", r.URL.Path)
			assert.Equal(t, "token test-token", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var body map[string]string
			err := json.NewDecoder(r.Body).Decode(&body)
			require.NoError(t, err)
			assert.Equal(t, "squash", body["merge_method"])

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		oldClient := githubutils.DefaultClient
		oldBaseURL := githubutils.BaseURL
		githubutils.DefaultClient = server.Client()
		githubutils.BaseURL = server.URL
		defer func() {
			githubutils.DefaultClient = oldClient
			githubutils.BaseURL = oldBaseURL
		}()

		err := githubutils.MergePullRequest("test-token", "owner", "repo", 123, "squash")
		assert.NoError(t, err)
	})

	t.Run("merge not allowed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}))
		defer server.Close()

		oldClient := githubutils.DefaultClient
		oldBaseURL := githubutils.BaseURL
		githubutils.DefaultClient = server.Client()
		githubutils.BaseURL = server.URL
		defer func() {
			githubutils.DefaultClient = oldClient
			githubutils.BaseURL = oldBaseURL
		}()

		err := githubutils.MergePullRequest("test-token", "owner", "repo", 123, "squash")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "merge is not allowed by GitHub")
	})

	t.Run("api error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		oldClient := githubutils.DefaultClient
		oldBaseURL := githubutils.BaseURL
		githubutils.DefaultClient = server.Client()
		githubutils.BaseURL = server.URL
		defer func() {
			githubutils.DefaultClient = oldClient
			githubutils.BaseURL = oldBaseURL
		}()

		err := githubutils.MergePullRequest("test-token", "owner", "repo", 123, "squash")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "GitHub API returned status 401")
	})
}

func TestFindRepoOwnerAndName(t *testing.T) {
	t.Run("https url", func(t *testing.T) {
		owner, repo, err := githubutils.FindRepoOwnerAndName()
		if err != nil {
			t.Skip("Not in a git repository")
		}
		assert.NotEmpty(t, owner)
		assert.NotEmpty(t, repo)
	})
}

func TestGetPullRequestTemplate(t *testing.T) {
	t.Run("template found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/repos/owner/repo/contents/.github/PULL_REQUEST_TEMPLATE.md", r.URL.Path)
			assert.Equal(t, "token test-token", r.Header.Get("Authorization"))

			// Base64 encoded "# Pull Request Template"
			response := map[string]string{
				"content":  "IyBQdWxsIFJlcXVlc3QgVGVtcGxhdGU=",
				"encoding": "base64",
			}
			err := json.NewEncoder(w).Encode(response)
			require.NoError(t, err)
		}))
		defer server.Close()

		oldClient := githubutils.DefaultClient
		oldBaseURL := githubutils.BaseURL
		githubutils.DefaultClient = server.Client()
		githubutils.BaseURL = server.URL
		defer func() {
			githubutils.DefaultClient = oldClient
			githubutils.BaseURL = oldBaseURL
		}()

		template, err := githubutils.GetPullRequestTemplate("test-token", "owner", "repo")
		assert.NoError(t, err)
		assert.Equal(t, "# Pull Request Template", template)
	})

	t.Run("no template found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		oldClient := githubutils.DefaultClient
		oldBaseURL := githubutils.BaseURL
		githubutils.DefaultClient = server.Client()
		githubutils.BaseURL = server.URL
		defer func() {
			githubutils.DefaultClient = oldClient
			githubutils.BaseURL = oldBaseURL
		}()

		template, err := githubutils.GetPullRequestTemplate("test-token", "owner", "repo")
		assert.NoError(t, err)
		assert.Empty(t, template)
	})
}
