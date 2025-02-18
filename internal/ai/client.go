package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// ConfigGetter is an interface for getting config values
type ConfigGetter interface {
	Get(key string, required bool) string
}

// configAdapter adapts a config.Get function to the ConfigGetter interface
type configAdapter struct {
	getFn func(key string, useLocal bool) string
}

func (c *configAdapter) Get(key string, required bool) string {
	return c.getFn(key, false) // Always use global config for AI settings
}

// NewConfigAdapter creates a new ConfigGetter from a config.Get function
func NewConfigAdapter(getFn func(key string, useLocal bool) string) ConfigGetter {
	return &configAdapter{getFn: getFn}
}

// Client is used to send requests to an AI provider following the OpenAI Chat API spec.
type Client struct {
	BaseURL    string
	APIKey     string
	Model      string
	config     ConfigGetter
	httpClient *http.Client
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
func NewClient(baseURL string, config ConfigGetter) *Client {
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

	// Check for API key in order of priority:
	// 1. Local config (if in a git repo)
	// 2. Global config
	// 3. Environment variable
	apiKey := config.Get("ai.api_key", true) // Try local config first
	if apiKey == "" {
		apiKey = config.Get("ai.api_key", false) // Try global config
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY") // Finally, try environment
		}
	}

	return &Client{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		Model:      model,
		config:     config,
		httpClient: &http.Client{},
	}
}

// SetHTTPClient sets a custom HTTP client for testing
func (c *Client) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

// GenerateCommitMessage sends the diff and a prompt to the AI provider and returns a commit message.
func (c *Client) GenerateCommitMessage(diff string) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("AI features require an API key. You can set it by either:\n" +
			"1. Setting the OPENAI_API_KEY environment variable\n" +
			"2. Running: sage config set ai.api_key <your-api-key>\n" +
			"3. If using a different AI provider, also set the base URL: sage config set ai.base_url <api-url>\n\n" +
			"Note: If you've already set the API key and are seeing this error, try setting it again as there might have been an issue with the encryption.")
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("invalid API key or unauthorized request. Please check your API key and provider settings:\n"+
			"• Current API URL: %s\n"+
			"• To update API key: sage config set ai.api_key <your-api-key>\n"+
			"• To update API URL: sage config set ai.base_url <api-url>\n"+
			"Provider error: %s", c.BaseURL, string(bodyBytes))
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed (status %d). Provider response: %s", resp.StatusCode, string(bodyBytes))
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

// GeneratePRDescription sends the diff and commits to generate a PR description
func (c *Client) GeneratePRDescription(commits, diff string) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("API key not found. Set OPENAI_API_KEY environment variable or configure in .sage/config.toml")
	}

	prompt := fmt.Sprintf(`You are a technical writer creating a comprehensive pull request description. Analyze the following commits and code changes to create a detailed, well-structured PR description.

Guidelines:
1. Structure the description with these sections:
   ## Summary
   - A clear, concise overview of the main changes (2-3 sentences)
   - Focus on the WHAT and WHY, not the how
   - Use business/feature-oriented language, not technical details
   
   ## Changes
   - Group changes by type (e.g., Features, Bug Fixes, Refactoring)
   - Use bullet points for better readability
   - Include relevant technical details but stay concise
   - Highlight important architectural decisions or trade-offs
   
   ## Testing
   - List specific test cases added/modified
   - Describe manual testing performed
   - Note areas that need careful review
   
   ## Breaking Changes
   - Only include if there are breaking changes
   - Clearly explain what breaks and why
   - Provide migration steps if applicable

2. Writing Style:
   - Use clear, professional language
   - Be concise but informative
   - Use active voice
   - Keep technical jargon to a minimum unless necessary
   - Use proper markdown formatting

3. Focus on:
   - Impact and value of the changes
   - Key technical decisions and their rationale
   - Potential risks or areas needing attention
   - User-facing changes (if any)

Commits:
%s

Changes:
%s

Generate a PR description following the above structure and guidelines. Use proper markdown formatting.`, commits, diff)

	// Build the request
	reqBody := GenerateRequest{
		Model: c.Model,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are a technical writer that creates clear, comprehensive pull request descriptions. Focus on clarity, completeness, and proper structure.",
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

	resp, err := c.httpClient.Do(req)
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

	// Get the generated description
	description := genResp.Choices[0].Message.Content

	// Clean up any potential HTML-like tags
	description = strings.ReplaceAll(description, "<", "\\<")
	description = strings.ReplaceAll(description, ">", "\\>")

	// Ensure proper markdown formatting
	if !strings.HasPrefix(description, "## Summary") {
		description = "## Summary\n" + description
	}

	return strings.TrimSpace(description), nil
}

// GeneratePRLabels sends the diff and commits to generate PR labels
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

// GeneratePRTitle sends the diff and commits to generate a PR title
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
