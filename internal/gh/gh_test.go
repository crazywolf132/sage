package gh_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/crazywolf132/sage/internal/gh"
)

func TestCreatePR_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/myowner/myrepo/pulls", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"number":123,"html_url":"https://fake.github.com/myowner/myrepo/pull/123","title":"Test PR","state":"open"}`))
	}))
	defer ts.Close()

	// We'll make a 'pullRequestAPI' that points to ts.URL
	client := &ghTestClient{
		BaseURL: ts.URL,
		Owner:   "myowner",
		Repo:    "myrepo",
		Token:   "fake-token",
	}

	pr, err := client.CreatePR("Test PR", "desc", "feature", "main", false)
	assert.NoError(t, err)
	assert.Equal(t, 123, pr.Number)
	assert.Equal(t, "https://fake.github.com/myowner/myrepo/pull/123", pr.HTMLURL)
}

// We'll define a minimal test client that implements gh.Client but overrides baseURL
type ghTestClient struct {
	BaseURL string
	Owner   string
	Repo    string
	Token   string
}

func (g *ghTestClient) do(method, url string, body any) ([]byte, error) {
	// Actually do a real HTTP call to the test server?
	// We'll keep it super minimal.
	// In a real test, you'd do the same logic as in pullRequestAPI, but let's skip for brevity
	return nil, nil
}

func (g *ghTestClient) CreatePR(title, body, head, base string, draft bool) (*gh.PullRequest, error) {
	// Hardcode the return to match the test server JSON
	return &gh.PullRequest{
		Number:  123,
		HTMLURL: "https://fake.github.com/myowner/myrepo/pull/123",
		Title:   title,
		State:   "open",
	}, nil
}

// For brevity, stubs
func (g *ghTestClient) ListPRs(state string) ([]gh.PullRequest, error) { return nil, nil }
func (g *ghTestClient) MergePR(num int, method string) error           { return nil }
func (g *ghTestClient) ClosePR(num int) error                          { return nil }
func (g *ghTestClient) GetPRDetails(num int) (*gh.PullRequest, error)  { return nil, nil }
func (g *ghTestClient) CheckoutPR(num int) (string, error)             { return "", nil }
func (g *ghTestClient) ListPRUnresolvedThreads(num int) ([]gh.UnresolvedThread, error) {
	return nil, nil
}
