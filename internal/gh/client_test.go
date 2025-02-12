package gh

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"
)

// mockExecCommand is used to mock exec.Command during tests
var mockExecCommand func(string, ...string) *exec.Cmd

func init() {
	// Replace the exec.Command with our mock
	execCommand = exec.Command
}

// Helper function to temporarily replace exec.Command with a mock
func withMockExec(mock func(string, ...string) *exec.Cmd, f func()) {
	oldExec := execCommand
	execCommand = mock
	defer func() { execCommand = oldExec }()
	f()
}

// Mock command that always fails
func mockCommandError(name string, args ...string) *exec.Cmd {
	return exec.Command("false")
}

func TestNewClient(t *testing.T) {
	// Save original environment
	origToken := os.Getenv("SAGE_GITHUB_TOKEN")
	origGHToken := os.Getenv("GITHUB_TOKEN")
	defer func() {
		os.Setenv("SAGE_GITHUB_TOKEN", origToken)
		os.Setenv("GITHUB_TOKEN", origGHToken)
	}()

	tests := []struct {
		name      string
		sageToken string
		ghToken   string
		wantToken string
		mockExec  func(string, ...string) *exec.Cmd
	}{
		{
			name:      "SAGE_GITHUB_TOKEN takes precedence",
			sageToken: "sage-token",
			ghToken:   "gh-token",
			wantToken: "sage-token",
			mockExec:  mockCommandError, // Should not be called
		},
		{
			name:      "Falls back to GITHUB_TOKEN",
			sageToken: "",
			ghToken:   "gh-token",
			wantToken: "gh-token",
			mockExec:  mockCommandError, // Should not be called
		},
		{
			name:      "No tokens",
			sageToken: "",
			ghToken:   "",
			wantToken: "",
			mockExec:  mockCommandError, // Force gh CLI check to fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("SAGE_GITHUB_TOKEN", tt.sageToken)
			os.Setenv("GITHUB_TOKEN", tt.ghToken)

			withMockExec(tt.mockExec, func() {
				client := NewClient()
				if c, ok := client.(*pullRequestAPI); ok {
					if c.token != tt.wantToken {
						t.Errorf("NewClient() token = %v, want %v", c.token, tt.wantToken)
					}
				} else {
					t.Error("NewClient() did not return *pullRequestAPI")
				}
			})
		})
	}
}

func TestCreatePR(t *testing.T) {
	tests := []struct {
		name       string
		title      string
		body       string
		head       string
		base       string
		draft      bool
		wantErr    bool
		statusCode int
		response   *PullRequest
	}{
		{
			name:       "Successful PR creation",
			title:      "Test PR",
			body:       "Test body",
			head:       "feature",
			base:       "main",
			draft:      false,
			wantErr:    false,
			statusCode: http.StatusOK,
			response: &PullRequest{
				Number:  1,
				Title:   "Test PR",
				HTMLURL: "https://github.com/test/repo/pull/1",
			},
		},
		{
			name:       "Invalid branch",
			title:      "Test PR",
			body:       "Test body",
			head:       "nonexistent",
			base:       "main",
			statusCode: http.StatusUnprocessableEntity,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.Header.Get("Authorization") != "token test-token" {
					t.Errorf("Expected Authorization header")
				}

				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			client := &pullRequestAPI{
				token:   "test-token",
				client:  http.DefaultClient,
				owner:   "test",
				repo:    "repo",
				baseURL: server.URL,
			}

			pr, err := client.CreatePR(tt.title, tt.body, tt.head, tt.base, tt.draft)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreatePR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && pr.Number != tt.response.Number {
				t.Errorf("CreatePR() got PR number %v, want %v", pr.Number, tt.response.Number)
			}
		})
	}
}

func TestListPRs(t *testing.T) {
	tests := []struct {
		name       string
		state      string
		wantErr    bool
		statusCode int
		response   []PullRequest
	}{
		{
			name:       "List open PRs",
			state:      "open",
			wantErr:    false,
			statusCode: http.StatusOK,
			response: []PullRequest{
				{Number: 1, Title: "PR 1", State: "open"},
				{Number: 2, Title: "PR 2", State: "open"},
			},
		},
		{
			name:       "List closed PRs",
			state:      "closed",
			wantErr:    false,
			statusCode: http.StatusOK,
			response: []PullRequest{
				{Number: 3, Title: "PR 3", State: "closed"},
			},
		},
		{
			name:       "API error",
			state:      "open",
			wantErr:    true,
			statusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				if state := r.URL.Query().Get("state"); state != tt.state {
					t.Errorf("Expected state %s, got %s", tt.state, state)
				}

				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			client := &pullRequestAPI{
				token:   "test-token",
				client:  http.DefaultClient,
				owner:   "test",
				repo:    "repo",
				baseURL: server.URL,
			}

			prs, err := client.ListPRs(tt.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListPRs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && len(prs) != len(tt.response) {
				t.Errorf("ListPRs() got %d PRs, want %d", len(prs), len(tt.response))
			}
		})
	}
}

func TestGetPRDetails(t *testing.T) {
	tests := []struct {
		name       string
		prNumber   int
		wantErr    bool
		statusCode int
		response   *PullRequest
	}{
		{
			name:       "Get existing PR",
			prNumber:   1,
			wantErr:    false,
			statusCode: http.StatusOK,
			response: &PullRequest{
				Number:  1,
				Title:   "Test PR",
				State:   "open",
				HTMLURL: "https://github.com/test/repo/pull/1",
				Head: struct {
					Ref string "json:\"ref\""
				}{
					Ref: "feature",
				},
				Base: struct {
					Ref string "json:\"ref\""
				}{
					Ref: "main",
				},
			},
		},
		{
			name:       "PR not found",
			prNumber:   999,
			wantErr:    true,
			statusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			client := &pullRequestAPI{
				token:   "test-token",
				client:  http.DefaultClient,
				owner:   "test",
				repo:    "repo",
				baseURL: server.URL,
			}

			pr, err := client.GetPRDetails(tt.prNumber)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPRDetails() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if pr.Number != tt.response.Number {
					t.Errorf("GetPRDetails() got PR number %v, want %v", pr.Number, tt.response.Number)
				}
				if pr.Head.Ref != tt.response.Head.Ref {
					t.Errorf("GetPRDetails() got Head.Ref %v, want %v", pr.Head.Ref, tt.response.Head.Ref)
				}
			}
		})
	}
}

func TestListPRUnresolvedThreads(t *testing.T) {
	tests := []struct {
		name       string
		prNumber   int
		wantErr    bool
		statusCode int
		response   []struct {
			ID        int       `json:"id"`
			Body      string    `json:"body"`
			Path      string    `json:"path"`
			Line      int       `json:"line"`
			CreatedAt time.Time `json:"created_at"`
			User      struct {
				Login string `json:"login"`
			} `json:"user"`
			DiffHunk string `json:"diff_hunk"`
		}
		wantThreads []UnresolvedThread
	}{
		{
			name:       "PR with unresolved threads",
			prNumber:   1,
			wantErr:    false,
			statusCode: http.StatusOK,
			response: []struct {
				ID        int       `json:"id"`
				Body      string    `json:"body"`
				Path      string    `json:"path"`
				Line      int       `json:"line"`
				CreatedAt time.Time `json:"created_at"`
				User      struct {
					Login string `json:"login"`
				} `json:"user"`
				DiffHunk string `json:"diff_hunk"`
			}{
				{
					ID:   1,
					Body: "Test comment",
					Path: "test.go",
					Line: 10,
					User: struct {
						Login string `json:"login"`
					}{
						Login: "testuser",
					},
					DiffHunk: "test diff",
				},
			},
			wantThreads: []UnresolvedThread{
				{
					Path:        "test.go",
					Line:        10,
					CodeContext: "test diff",
					Comments: []Comment{
						{
							User: "testuser",
							Body: "Test comment",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			client := &pullRequestAPI{
				token:   "test-token",
				client:  http.DefaultClient,
				owner:   "test",
				repo:    "repo",
				baseURL: server.URL,
			}

			threads, err := client.ListPRUnresolvedThreads(tt.prNumber)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListPRUnresolvedThreads() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if len(threads) != len(tt.wantThreads) {
					t.Errorf("ListPRUnresolvedThreads() got %d threads, want %d", len(threads), len(tt.wantThreads))
				}
				for i, thread := range threads {
					want := tt.wantThreads[i]
					if thread.Path != want.Path {
						t.Errorf("Thread[%d].Path = %v, want %v", i, thread.Path, want.Path)
					}
					if thread.Line != want.Line {
						t.Errorf("Thread[%d].Line = %v, want %v", i, thread.Line, want.Line)
					}
					if thread.CodeContext != want.CodeContext {
						t.Errorf("Thread[%d].CodeContext = %v, want %v", i, thread.CodeContext, want.CodeContext)
					}
				}
			}
		})
	}
}

func TestGetLatestRelease(t *testing.T) {
	tests := []struct {
		name       string
		wantErr    bool
		statusCode int
		response   struct {
			TagName string `json:"tag_name"`
		}
		wantVersion string
	}{
		{
			name:       "Latest release exists",
			wantErr:    false,
			statusCode: http.StatusOK,
			response: struct {
				TagName string `json:"tag_name"`
			}{
				TagName: "v1.0.0",
			},
			wantVersion: "1.0.0",
		},
		{
			name:       "No releases",
			wantErr:    true,
			statusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			client := &pullRequestAPI{
				token:   "test-token",
				client:  http.DefaultClient,
				owner:   "test",
				repo:    "repo",
				baseURL: server.URL,
			}

			version, err := client.GetLatestRelease()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLatestRelease() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && version != tt.wantVersion {
				t.Errorf("GetLatestRelease() = %v, want %v", version, tt.wantVersion)
			}
		})
	}
}
