package ai

import (
	"context"
	"fmt"
	"strings"
)

// LLM represents a language model interface
type LLM interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

// Client is an AI client that provides various AI-powered features
type Client struct {
	llm LLM
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

// NewClient creates a new AI client with the given LLM
func NewClient(llm LLM) *Client {
	return &Client{
		llm: llm,
	}
}

// GenerateCommitMessage sends the diff and a prompt to the AI provider and returns a commit message.
func (c *Client) GenerateCommitMessage(diff string) (string, error) {
	prompt := fmt.Sprintf(`You are a helpful git commit message generator. Your task is to analyze the following code changes and generate a clear, meaningful commit message that follows the Conventional Commits specification.

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

Code changes to analyze:
%s

Respond with ONLY the commit message, no additional text or formatting.`, diff)

	return c.llm.Complete(context.Background(), prompt)
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

	return c.llm.Complete(context.Background(), prompt)
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

	response, err := c.llm.Complete(context.Background(), prompt)
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

	return c.llm.Complete(context.Background(), prompt)
}
