package ai

import (
	"context"
	"fmt"
	"strings"
)

// GenerateBranchName generates a Git branch name from a description
func (c *Client) GenerateBranchName(description string) (string, error) {
	prompt := fmt.Sprintf(`Generate a Git branch name for the following feature description:
"%s"

Requirements:
- Use kebab-case (lowercase with hyphens)
- Start with a type prefix (feat, fix, chore, docs, etc.)
- Keep it concise but descriptive
- No special characters except hyphens
- Maximum length of 50 characters

Example outputs:
- feat-add-user-authentication
- fix-memory-leak-in-worker
- chore-update-dependencies
- docs-api-documentation

Branch name:`, description)

	resp, err := c.llm.Complete(context.Background(), prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate branch name: %w", err)
	}

	// Clean up the response
	branchName := strings.TrimSpace(resp)
	branchName = strings.ToLower(branchName)
	branchName = strings.ReplaceAll(branchName, " ", "-")

	// Remove any special characters except hyphens
	var result strings.Builder
	for _, ch := range branchName {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
			result.WriteRune(ch)
		}
	}

	return result.String(), nil
}
