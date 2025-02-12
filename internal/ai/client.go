package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/crazywolf132/sage/internal/config"
)

// Client is used to send requests to an AI provider following the OpenAI Chat API spec.
type Client struct {
	BaseURL string
	APIKey  string
	Model   string
}

// GenerateRequest matches the minimal payload expected by the Chat API.
type GenerateRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// Message is a single message in the conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GenerateResponse is the full response from the API.
type GenerateResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice is one possible completion.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage provides token usage information.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewClient creates a new Ollama client.
func NewClient(baseURL string) *Client {
	model := config.Get("ai.model", false)
	if model == "" {
		model = "gpt-4o"
	}

	// Use passed baseURL first, then config, then default
	if baseURL == "" {
		baseURL = config.Get("ai.base_url", false)
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = config.Get("ai.api_key", false)
	}

	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
	}
}

// GenerateCommitMessage sends the diff and a prompt to the AI provider and returns a commit message.
func (c *Client) GenerateCommitMessage(diff string) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("API key not found. Set OPENAI_API_KEY environment variable or configure in .sage/config.toml")
	}

	// OpenAI (or compatible) providers have a maximum allowed content length.
	const maxAllowed = 1048576

	// Define the static parts of the prompt.
	staticPrefix := `You are a helpful git commit message generator. Your task is to analyze the following code changes and generate a clear, meaningful commit message that follows the Conventional Commits specification.

Guidelines:
1. Use one of these types:
   - feat: A new feature
   - fix: A bug fix
   - docs: Documentation changes
   - style: Code style changes (formatting, missing semi-colons, etc)
   - refactor: Code changes that neither fix a bug nor add a feature
   - test: Adding or modifying tests
   - ci: Changes to CI/CD configuration and scripts
   - chore: Changes to build process or auxiliary tools

2. Format: <type>: <description>
   Examples:
   - feat: add user authentication system
   - fix: resolve null pointer in data processing
   - ci: update GitHub Actions workflow

3. Analyze the diff carefully:
   - Look for function/method additions or modifications
   - Identify bug fixes from error handling changes
   - Note any test additions or modifications
   - Consider impact on existing functionality
   - Changes in .github/workflows/ directory should use 'ci' type
   - Changes to CI/CD pipeline configurations should use 'ci' type

4. Keep the message:
   - Concise but informative (ideally under 72 characters)
   - Focused on WHAT changed and WHY
   - In imperative mood ("add" not "added")
   - Without unnecessary technical details

Code changes to analyze:
`
	staticSuffix := `

Respond with ONLY the commit message, no additional text or formatting.`

	// Calculate the maximum allowed length for the diff portion.
	allowedDiffLength := maxAllowed - (len(staticPrefix) + len(staticSuffix))
	if len(diff) > allowedDiffLength {
		diff = diff[:allowedDiffLength] + "\n[diff truncated]"
	}

	// Construct the full prompt.
	prompt := staticPrefix + diff + staticSuffix

	// Build the request with two messages.
	reqBody := GenerateRequest{
		Model: c.Model,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are a helpful git commit message generator that follows the Conventional Commits specification.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", c.BaseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d. Response: %s", resp.StatusCode, string(bodyBytes))
	}

	var genResp GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(genResp.Choices) == 0 {
		return "", fmt.Errorf("no response generated")
	}

	if genResp.Choices[0].FinishReason != "stop" {
		return "", fmt.Errorf("incomplete response: %s", genResp.Choices[0].FinishReason)
	}

	// Assemble the commit message from the response.
	response := genResp.Choices[0].Message.Content
	for {
		start := strings.Index(response, "<")
		if start == -1 {
			break
		}
		end := strings.Index(response[start:], ">")
		if end == -1 {
			break
		}
		response = response[:start] + response[start+end+1:]
	}

	return strings.TrimSpace(response), nil
}

func (c *Client) GeneratePRDescription(commits, diff string) (string, error) {
	prompt := fmt.Sprintf(`You are a technical writer creating a comprehensive pull request description. Analyze the following commits and code changes to create a detailed, well-structured PR description.

Guidelines:
1. Structure the description with these sections:
   ## Summary
   - High-level overview of the changes
   - The problem being solved or feature being added
   
   ## Implementation Details
   - Key technical changes and design decisions
   - Important code changes or new components
   - Any dependencies added or modified
   
   ## Testing
   - How the changes were tested
   - Any new tests added
   - Areas that need careful review/testing
   
   ## Breaking Changes
   - List any breaking changes (if applicable)
   - Migration steps (if needed)
   
   ## Related Issues
   - Reference any related issues or tickets (if apparent from commits)

2. Focus on:
   - The WHY behind the changes
   - Key technical decisions
   - Impact on the codebase
   - Areas that need reviewer attention

3. Format:
   - Use clear markdown formatting
   - Keep sections concise but informative
   - Use bullet points for better readability

Commits:
%s

Changes:
%s

Generate a PR description following the above structure and guidelines. Use proper markdown formatting.`, commits, diff)

	return c.GenerateCommitMessage(prompt)
}

func (c *Client) GeneratePRLabels(commits, diff string) ([]string, error) {
	prompt := fmt.Sprintf(`Based on these changes, suggest appropriate GitHub PR labels from this list ONLY:
- feature
- bug
- documentation
- enhancement
- refactor
- breaking
- dependencies
- testing

Commits:
%s

Changes:
%s

Return ONLY the exact label names from the list above, separated by commas. Do not add any new labels.`, commits, diff)

	response, err := c.GenerateCommitMessage(prompt)
	if err != nil {
		return nil, err
	}

	// Parse and clean labels
	labels := strings.Split(strings.TrimSpace(response), ",")
	validLabels := map[string]bool{
		"feature":       true,
		"bug":           true,
		"documentation": true,
		"enhancement":   true,
		"refactor":      true,
		"breaking":      true,
		"dependencies":  true,
		"testing":       true,
	}

	// Filter out invalid labels
	var cleanedLabels []string
	for _, label := range labels {
		label = strings.TrimSpace(label)
		if validLabels[label] {
			cleanedLabels = append(cleanedLabels, label)
		}
	}

	return cleanedLabels, nil
}

func (c *Client) GeneratePRTitle(commits, diff string) (string, error) {
	prompt := fmt.Sprintf(`Based on these commits and changes, generate a PR title that follows the Conventional Commits specification.

The title MUST follow this format:
type(optional-scope): description

Where type is one of:
- feat: A new feature
- fix: A bug fix
- docs: Documentation changes
- style: Code style changes (formatting, etc)
- refactor: Code changes that neither fix a bug nor add a feature
- test: Adding or modifying tests
- chore: Changes to build process or auxiliary tools

Guidelines:
1. The description should be clear and concise (under 72 chars total)
2. Use imperative mood ("add" not "added")
3. Focus on the main change
4. If there are breaking changes, add "!" after the type (e.g., "feat!: breaking change")

Commits:
%s

Changes:
%s

Return only the conventional commit title, no additional text or formatting.`, commits, diff)

	return c.GenerateCommitMessage(prompt)
}
