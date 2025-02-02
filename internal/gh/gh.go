package gh

import (
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Client is the interface for GitHub operations
type Client interface {
	CreatePR(title, body, head, base string, draft bool) (*PullRequest, error)
	ListPRs(state string) ([]PullRequest, error)
	MergePR(num int, method string) error
	ClosePR(num int) error
	GetPRDetails(num int) (*PullRequest, error)
	CheckoutPR(num int) (string, error)
	ListPRUnresolvedThreads(prNum int) ([]UnresolvedThread, error)
	GetPRTemplate() (string, error)
	AddLabels(prNumber int, labels []string) error
	RequestReviewers(prNumber int, reviewers []string) error
	GetPRForBranch(branchName string) (*PullRequest, error)
}

// pullRequestAPI is a minimal data holder for GH API calls
type pullRequestAPI struct {
	token  string
	client *http.Client
	owner  string
	repo   string
}

// PullRequest is the domain object representing a PR
type PullRequest struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	State   string `json:"state"`
	HTMLURL string `json:"html_url"`
	Draft   bool   `json:"draft"`
	Merged  bool   `json:"merged"`
	Head    struct {
		Ref string `json:"ref"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
	Reviews  []Review        `json:"reviews"`
	Checks   []Check         `json:"checks"`
	Timeline []TimelineEvent `json:"timeline"`
}

type Review struct {
	State string `json:"state"`
	User  struct {
		Login string `json:"login"`
	} `json:"user"`
}

type Check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type TimelineEvent struct {
	Event     string    `json:"event"`
	CreatedAt time.Time `json:"created_at"`
	Message   string    `json:"message"`
	SHA       string    `json:"sha"`
	Actor     struct {
		Login string `json:"login"`
	} `json:"actor"`
}

// UnresolvedThread is a minimal structure for unresolved PR comment threads
type UnresolvedThread struct {
	Path     string
	Line     int
	Comments []Comment
}

// Comment is a snippet representing a single comment in a thread
type Comment struct {
	User string
	Body string
}

// NewClient creates a new GitHub client, discovering token from env or gh CLI,
// and discovering owner+repo from the local Git config (git remote get-url origin).
func NewClient() Client {
	tok := discoverToken()
	ow, rp := discoverOwnerRepo()

	return &pullRequestAPI{
		token:  tok,
		client: http.DefaultClient,
		owner:  ow,
		repo:   rp,
	}
}

// discoverToken tries environment variables or "gh auth token".
func discoverToken() string {
	if t := os.Getenv("SAGE_GITHUB_TOKEN"); t != "" {
		return t
	}
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		return t
	}

	// fallback: gh CLI
	out, err := exec.Command("gh", "auth", "token").Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return ""
}

// discoverOwnerRepo parses "git remote get-url origin" to find "owner", "repo"
func discoverOwnerRepo() (string, string) {
	remote, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return "unknown", "unknown"
	}
	url := strings.TrimSpace(string(remote))
	url = strings.TrimSuffix(url, ".git")

	// Examples:
	// https://github.com/owner/repo
	// git@github.com:owner/repo.git
	// We'll do a naive approach:
	url = strings.Replace(url, "git@github.com:", "", 1)
	url = strings.Replace(url, "https://github.com/", "", 1)
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "unknown", "unknown"
	}
	return parts[0], strings.TrimSuffix(parts[1], ".git")
}
