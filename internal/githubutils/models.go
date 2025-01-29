package githubutils

import (
	"time"
)

// CreatePRParams is what we send to GitHub to create a new PR.
type CreatePRParams struct {
	Title string `json:"title"`
	Head  string `json:"head"`
	Base  string `json:"base"`
	Body  string `json:"body,omitempty"`
	Draft bool   `json:"draft,omitempty"`
}

// PullRequest represents a subset of GitHub's PR object (expand as needed).
type PullRequest struct {
	Number  int    `json:"number"`
	HTMLURL string `json:"html_url"`
	Title   string `json:"title"`
	State   string `json:"state"`
	Body    string `json:"body"`
	Draft   bool   `json:"draft"`
	Head    struct {
		Ref string `json:"ref"`
	} `json:"head"`
}

// PullRequestDetails represents detailed information about a PR
type PullRequestDetails struct {
	Number  int    `json:"number"`
	HTMLURL string `json:"html_url"`
	Title   string `json:"title"`
	State   string `json:"state"`
	Body    string `json:"body"`
	Draft   bool   `json:"draft"`
	Merged  bool   `json:"merged"`
	Head    struct {
		Ref string `json:"ref"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
	Reviews  []PRReview `json:"reviews"`
	Checks   []PRCheck  `json:"check_runs"`
	Timeline []PREvent  `json:"timeline_events"`
}

// PRReview represents a review on a pull request
type PRReview struct {
	State string `json:"state"`
	User  struct {
		Login string `json:"login"`
	} `json:"user"`
}

// PRCheck represents a CI check on a pull request
type PRCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// PREvent represents a timeline event on a pull request
type PREvent struct {
	Event     string    `json:"event"`
	Actor     string    `json:"actor"`
	CreatedAt time.Time `json:"created_at"`
}
