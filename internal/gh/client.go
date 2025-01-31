package gh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
)

const baseURL = "https://api.github.com"

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

// GetPRDetails does GET /repos/:owner/:repo/pulls/:pull_number
func (p *pullRequestAPI) GetPRDetails(num int) (*PullRequest, error) {
	u := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", baseURL, p.owner, p.repo, num)
	data, err := p.do("GET", u, nil)
	if err != nil {
		return nil, err
	}
	var pr PullRequest
	if e := json.Unmarshal(data, &pr); e != nil {
		return nil, e
	}
	return &pr, nil
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
		ID   int    `json:"id"`
		Body string `json:"body"`
		Path string `json:"path"`
		Line int    `json:"line"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
	}
	if e := json.Unmarshal(data, &raw); e != nil {
		return nil, e
	}

	m := make(map[string][]Comment)
	for _, r := range raw {
		// assume they're all unresolved
		key := r.Path + ":" + strconv.Itoa(r.Line)
		m[key] = append(m[key], Comment{
			User: r.User.Login,
			Body: r.Body,
		})
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
			Path:     pth,
			Line:     ln,
			Comments: cList,
		})
	}
	return results, nil
}

// runCmd is a helper to run shell commands.
func runCmd(prog string, args ...string) (string, error) {
	cmd := exec.Command(prog, args...)
	b, err := cmd.CombinedOutput()
	return string(b), err
}
