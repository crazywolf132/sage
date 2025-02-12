package gh

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://api.github.com"

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
	Path        string
	Line        int
	Comments    []Comment
	CodeContext string // The code snippet around the comment
}

// Comment is a snippet representing a single comment in a thread
type Comment struct {
	User string
	Body string
	Time time.Time // When the comment was made
}

// Client interface defines methods for interacting with GitHub
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
	GetLatestRelease() (string, error)
	UpdatePR(num int, pr *PullRequest) error
}

// TokenSource represents where the GitHub token was obtained from
type TokenSource struct {
	Token  string
	Source string
}

func (p *pullRequestAPI) do(method, url string, body any) ([]byte, error) {
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		buf = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, buf)
	if err != nil {
		return nil, err
	}
	if p.token != "" {
		req.Header.Set("Authorization", "token "+p.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API %s %s returned %d:\n%s",
			method, url, resp.StatusCode, string(msg))
	}
	return io.ReadAll(resp.Body)
}

// CreatePR does a POST /repos/:owner/:repo/pulls
func (p *pullRequestAPI) CreatePR(title, body, head, base string, draft bool) (*PullRequest, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls", baseURL, p.owner, p.repo)
	payload := map[string]any{
		"title": title,
		"body":  body,
		"head":  head,
		"base":  base,
		"draft": draft,
	}
	data, err := p.do("POST", url, payload)
	if err != nil {
		return nil, err
	}
	var pr PullRequest
	if e := json.Unmarshal(data, &pr); e != nil {
		return nil, e
	}
	return &pr, nil
}

// ListPRs does GET /repos/:owner/:repo/pulls?state=STATE
func (p *pullRequestAPI) ListPRs(state string) ([]PullRequest, error) {
	u := fmt.Sprintf("%s/repos/%s/%s/pulls?state=%s", baseURL, p.owner, p.repo, state)
	data, err := p.do("GET", u, nil)
	if err != nil {
		return nil, err
	}
	var prs []PullRequest
	if e := json.Unmarshal(data, &prs); e != nil {
		return nil, e
	}
	return prs, nil
}

// MergePR does PUT /repos/:owner/:repo/pulls/:pull_number/merge
func (p *pullRequestAPI) MergePR(num int, method string) error {
	u := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/merge", baseURL, p.owner, p.repo, num)
	payload := map[string]string{
		"merge_method": method, // "merge", "squash", or "rebase"
	}
	_, err := p.do("PUT", u, payload)
	return err
}

// ClosePR does PATCH /repos/:owner/:repo/pulls/:pull_number (state=closed)
func (p *pullRequestAPI) ClosePR(num int) error {
	u := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", baseURL, p.owner, p.repo, num)
	payload := map[string]string{
		"state": "closed",
	}
	_, err := p.do("PATCH", u, payload)
	return err
}

// GetPRDetails does GET /repos/:owner/:repo/pulls/:pull_number and fetches additional data
func (p *pullRequestAPI) GetPRDetails(num int) (*PullRequest, error) {
	// Get basic PR info
	u := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", baseURL, p.owner, p.repo, num)
	data, err := p.do("GET", u, nil)
	if err != nil {
		return nil, err
	}
	var pr PullRequest
	if e := json.Unmarshal(data, &pr); e != nil {
		return nil, e
	}

	// Get reviews
	reviews, err := p.getPRReviews(num)
	if err != nil {
		// Don't fail if we can't get reviews
		fmt.Printf("Warning: failed to get reviews: %v\n", err)
	}
	pr.Reviews = reviews

	// Get checks
	checks, err := p.getPRChecks(num)
	if err != nil {
		// Don't fail if we can't get checks
		fmt.Printf("Warning: failed to get checks: %v\n", err)
	}
	pr.Checks = checks

	// Get timeline
	timeline, err := p.getPRTimeline(num)
	if err != nil {
		// Don't fail if we can't get timeline
		fmt.Printf("Warning: failed to get timeline: %v\n", err)
	}
	pr.Timeline = timeline

	return &pr, nil
}

func (p *pullRequestAPI) getPRReviews(num int) ([]Review, error) {
	u := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/reviews", baseURL, p.owner, p.repo, num)
	data, err := p.do("GET", u, nil)
	if err != nil {
		return nil, err
	}
	var reviews []Review
	if e := json.Unmarshal(data, &reviews); e != nil {
		return nil, e
	}
	return reviews, nil
}

func (p *pullRequestAPI) getPRChecks(num int) ([]Check, error) {
	// First get the ref (SHA) for the PR head
	u := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", baseURL, p.owner, p.repo, num)
	data, err := p.do("GET", u, nil)
	if err != nil {
		return nil, err
	}
	var pr struct {
		Head struct {
			SHA string `json:"sha"`
		} `json:"head"`
	}
	if e := json.Unmarshal(data, &pr); e != nil {
		return nil, e
	}

	// Then get check runs for that SHA
	u = fmt.Sprintf("%s/repos/%s/%s/commits/%s/check-runs", baseURL, p.owner, p.repo, pr.Head.SHA)
	data, err = p.do("GET", u, nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		CheckRuns []Check `json:"check_runs"`
	}
	if e := json.Unmarshal(data, &resp); e != nil {
		return nil, e
	}
	return resp.CheckRuns, nil
}

func (p *pullRequestAPI) getPRTimeline(num int) ([]TimelineEvent, error) {
	u := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/commits", baseURL, p.owner, p.repo, num)
	data, err := p.do("GET", u, nil)
	if err != nil {
		return nil, err
	}
	var commits []struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message   string `json:"message"`
			Committer struct {
				Date time.Time `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
		Author struct {
			Login string `json:"login"`
		} `json:"author"`
	}
	if e := json.Unmarshal(data, &commits); e != nil {
		return nil, e
	}

	var timeline []TimelineEvent
	for _, c := range commits {
		timeline = append(timeline, TimelineEvent{
			Event:     "committed",
			CreatedAt: c.Commit.Committer.Date,
			Message:   c.Commit.Message,
			SHA:       c.SHA,
			Actor: struct {
				Login string `json:"login"`
			}{
				Login: c.Author.Login,
			},
		})
	}
	return timeline, nil
}

// CheckoutPR fetches the branch and switches locally
// We do "git fetch origin HEAD-REF" then create a local branch
func (p *pullRequestAPI) CheckoutPR(num int) (string, error) {
	pr, err := p.GetPRDetails(num)
	if err != nil {
		return "", err
	}
	branchName := pr.Head.Ref

	// fetch
	if out, err := runCmd("git", "fetch", "origin", branchName); err != nil {
		return "", fmt.Errorf("fetch error: %s\n%s", err, out)
	}
	// try "git switch -c BRANCH --track origin/BRANCH"
	if out, err := runCmd("git", "switch", "-c", branchName, "--track", "origin/"+branchName); err != nil {
		// if that fails, maybe the branch already exists:
		if strings.Contains(string(out), "already exists.") || strings.Contains(string(out), "already exists") {
			// do fallback
			if out2, err2 := runCmd("git", "switch", branchName); err2 != nil {
				return "", fmt.Errorf("switch error: %s\n%s", err2, out2)
			}
			// reset
			if out3, err3 := runCmd("git", "reset", "--hard", "origin/"+branchName); err3 != nil {
				return "", fmt.Errorf("reset error: %s\n%s", err3, out3)
			}
		} else {
			return "", fmt.Errorf("switch error: %s\n%s", err, out)
		}
	}

	return branchName, nil
}

// ListPRUnresolvedThreads checks review comments for prNumber
func (p *pullRequestAPI) ListPRUnresolvedThreads(prNum int) ([]UnresolvedThread, error) {
	u := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/comments", baseURL, p.owner, p.repo, prNum)
	data, err := p.do("GET", u, nil)
	if err != nil {
		return nil, err
	}
	var raw []struct {
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
	if e := json.Unmarshal(data, &raw); e != nil {
		return nil, e
	}

	m := make(map[string][]Comment)
	contexts := make(map[string]string)
	for _, r := range raw {
		// assume they're all unresolved
		key := r.Path + ":" + strconv.Itoa(r.Line)
		m[key] = append(m[key], Comment{
			User: r.User.Login,
			Body: r.Body,
			Time: r.CreatedAt,
		})
		// Store the code context if available
		if r.DiffHunk != "" {
			contexts[key] = r.DiffHunk
		}
	}

	var results []UnresolvedThread
	for k, cList := range m {
		parts := strings.Split(k, ":")
		pth := parts[0]
		var ln int
		if len(parts) == 2 {
			ln, _ = strconv.Atoi(parts[1])
		}
		results = append(results, UnresolvedThread{
			Path:        pth,
			Line:        ln,
			Comments:    cList,
			CodeContext: contexts[k],
		})
	}
	return results, nil
}

func (p *pullRequestAPI) GetPRTemplate() (string, error) {
	// Try multiple locations and formats for PR templates
	filesToCheck := []string{
		".github/pull_request_template.md",
		".github/PULL_REQUEST_TEMPLATE.md",
		".github/pull_request_template",
		".github/PULL_REQUEST_TEMPLATE",
		"docs/pull_request_template.md",
		"docs/PULL_REQUEST_TEMPLATE.md",
		".github/PULL_REQUEST_TEMPLATE/", // Directory-based templates
		"pull_request_template.md",
		"PULL_REQUEST_TEMPLATE.md",
	}

	// First try the directory-based template approach
	dirContent, err := p.getDirectoryContent(".github/PULL_REQUEST_TEMPLATE")
	if err == nil && len(dirContent) > 0 {
		// Use the first template found in the directory
		for _, file := range dirContent {
			if strings.HasSuffix(strings.ToLower(file.Name), ".md") {
				content, err := p.getContentFile(file.Path)
				if err == nil && content != "" {
					return content, nil
				}
			}
		}
	}

	// Then try individual files
	for _, f := range filesToCheck {
		content, err := p.getContentFile(f)
		if err == nil && content != "" {
			return content, nil
		}
	}

	return "", nil
}

func (p *pullRequestAPI) getDirectoryContent(path string) ([]struct {
	Name string `json:"name"`
	Path string `json:"path"`
}, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", baseURL, p.owner, p.repo, path)
	data, err := p.do("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var files []struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal(data, &files); err != nil {
		return nil, err
	}

	return files, nil
}

func (p *pullRequestAPI) getContentFile(path string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", baseURL, p.owner, p.repo, path)
	data, err := p.do("GET", url, nil)
	if err != nil {
		return "", err
	}
	var resp struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if e := json.Unmarshal(data, &resp); e != nil {
		return "", e
	}
	if resp.Encoding == "base64" && resp.Content != "" {
		decoded, e2 := decodeBase64(resp.Content)
		return decoded, e2
	}
	return "", nil
}

func (p *pullRequestAPI) AddLabels(prNumber int, labels []string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/labels", baseURL, p.owner, p.repo, prNumber)
	payload := map[string][]string{
		"labels": labels,
	}
	_, err := p.do("POST", url, payload)
	return err
}

func (p *pullRequestAPI) RequestReviewers(prNumber int, reviewers []string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/requested_reviewers", baseURL, p.owner, p.repo, prNumber)
	payload := map[string][]string{
		"reviewers": reviewers,
	}
	_, err := p.do("POST", url, payload)
	return err
}

// decodeBase64 is a minimal helper
func decodeBase64(s string) (string, error) {
	// If you want to use the built-in:
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// runCmd is a helper to run shell commands.
func runCmd(prog string, args ...string) (string, error) {
	cmd := exec.Command(prog, args...)
	b, err := cmd.CombinedOutput()
	return string(b), err
}

// GetPRForBranch returns the pull request for the given branch name.
// Returns nil if no PR exists for the branch.
func (p *pullRequestAPI) GetPRForBranch(branchName string) (*PullRequest, error) {
	// List open PRs for this branch
	u := fmt.Sprintf("%s/repos/%s/%s/pulls?state=open&head=%s:%s", baseURL, p.owner, p.repo, p.owner, branchName)
	data, err := p.do("GET", u, nil)
	if err != nil {
		return nil, err
	}
	var prs []PullRequest
	if e := json.Unmarshal(data, &prs); e != nil {
		return nil, e
	}
	if len(prs) == 0 {
		return nil, nil
	}
	return &prs[0], nil
}

// GetLatestRelease returns the latest release version from GitHub
func (p *pullRequestAPI) GetLatestRelease() (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", baseURL, p.owner, p.repo)
	data, err := p.do("GET", url, nil)
	if err != nil {
		return "", err
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(data, &release); err != nil {
		return "", err
	}

	// Remove 'v' prefix if present
	version := strings.TrimPrefix(release.TagName, "v")
	return version, nil
}

// NewClient creates a new GitHub client
func NewClient() Client {
	owner, repo := getOwnerAndRepo()
	if owner == "" || repo == "" {
		panic("Could not determine GitHub repository. Please ensure you have a valid git remote or set SAGE_GITHUB_OWNER and SAGE_GITHUB_REPO environment variables")
	}

	tokenSource := getToken()
	if tokenSource.Token == "" {
		panic(`GitHub token not found. Please either:
1. Set SAGE_GITHUB_TOKEN environment variable
2. Set GITHUB_TOKEN environment variable
3. Login with 'gh auth login' to use GitHub CLI authentication
	 `)
	}

	fmt.Printf("Using GitHub token from: %s\n", tokenSource.Source)

	return &pullRequestAPI{
		owner:  owner,
		repo:   repo,
		token:  tokenSource.Token,
		client: &http.Client{},
	}
}

// getToken returns the GitHub token from various sources
func getToken() TokenSource {
	// Try environment variables first
	token := os.Getenv("SAGE_GITHUB_TOKEN")
	if token != "" {
		return TokenSource{Token: token, Source: "SAGE_GITHUB_TOKEN environment variable"}
	}

	token = os.Getenv("GITHUB_TOKEN")
	if token != "" {
		return TokenSource{Token: token, Source: "GITHUB_TOKEN environment variable"}
	}

	// Try to get token from gh CLI
	ghToken, err := getGHCliToken()
	if err == nil && ghToken != "" {
		return TokenSource{Token: ghToken, Source: "GitHub CLI"}
	}

	return TokenSource{}
}

// getGHCliToken attempts to get the GitHub token from the gh CLI configuration
func getGHCliToken() (string, error) {
	// Try to get token using gh CLI
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err == nil {
		token := strings.TrimSpace(string(output))
		if token != "" {
			return token, nil
		}
	}

	return "", fmt.Errorf("no token found in gh CLI config")
}

// getOwnerAndRepo extracts the owner and repo from the git remote URL
func getOwnerAndRepo() (string, string) {
	// Try environment variables first
	owner := os.Getenv("SAGE_GITHUB_OWNER")
	repo := os.Getenv("SAGE_GITHUB_REPO")
	if owner != "" && repo != "" {
		return owner, repo
	}

	// Fall back to git remote
	cmd := exec.Command("git", "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return "", ""
	}

	url := strings.TrimSpace(string(out))
	url = strings.TrimSuffix(url, ".git")

	// Handle SSH URLs
	if strings.HasPrefix(url, "git@") {
		parts := strings.Split(strings.TrimPrefix(url, "git@github.com:"), "/")
		if len(parts) >= 2 {
			return parts[0], parts[1]
		}
		return "", ""
	}

	// Handle HTTPS URLs
	if strings.HasPrefix(url, "https://") {
		parts := strings.Split(strings.TrimPrefix(url, "https://github.com/"), "/")
		if len(parts) >= 2 {
			return parts[0], parts[1]
		}
	}

	return "", ""
}

// UpdatePR updates the specified pull request with new details
func (p *pullRequestAPI) UpdatePR(num int, pr *PullRequest) error {
	// Build the update request
	data := map[string]interface{}{
		"title": pr.Title,
		"body":  pr.Body,
		"draft": pr.Draft,
	}

	// Make the PATCH request to update the PR
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d", p.owner, p.repo, num)
	req, err := http.NewRequest("PATCH", url, strings.NewReader(jsonEncode(data)))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update PR: %s", resp.Status)
	}

	return nil
}

func jsonEncode(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}
