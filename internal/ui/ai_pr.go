package ui

import (
	"fmt"
	"strings"

	"github.com/crazywolf132/sage/internal/gh"
	"github.com/crazywolf132/sage/internal/git"
)

// GenerateAIPRContent uses git diff and commit history to generate PR content
func GenerateAIPRContent(g git.Service, ghc gh.Client) (PRForm, error) {
	form := PRForm{}

	// Get the current branch name
	branch, err := g.CurrentBranch()
	if err != nil {
		return form, fmt.Errorf("failed to get current branch: %w", err)
	}

	// Get the default branch
	defaultBranch, err := g.DefaultBranch()
	if err != nil {
		defaultBranch = "main" // fallback
	}

	// Get the diff between current branch and default branch
	// First get the diff of staged changes
	stagedDiff, err := g.GetDiff()
	if err != nil {
		return form, fmt.Errorf("failed to get staged changes: %w", err)
	}

	// Get commit messages for this branch only (since branching from default branch)
	commits, err := g.Log(fmt.Sprintf("%s..%s", defaultBranch, branch), 0, false, false)
	if err != nil {
		return form, fmt.Errorf("failed to get branch commit history: %w", err)
	}

	// Try to get PR template
	template, _ := ghc.GetPRTemplate()

	// Generate PR content using the collected information
	content, err := generatePRContent(GenerateInput{
		Branch:        branch,
		DefaultBranch: defaultBranch,
		Diff:          stagedDiff,
		Commits:       commits,
		Template:      template,
	})
	if err != nil {
		return form, fmt.Errorf("failed to generate content: %w", err)
	}

	form.Title = content.Title
	form.Body = content.Body
	form.Labels = content.Labels
	form.Base = defaultBranch

	// Show preview of generated content
	fmt.Printf("\n%s Generated PR Title: %s\n", Green("✓"), form.Title)
	preview := truncateBody(form.Body, 10, 80)
	fmt.Printf("\n%s Generated PR Description:\n%s\n\n", Green("✓"), preview)

	return form, nil
}

type GenerateInput struct {
	Branch        string
	DefaultBranch string
	Diff          string
	Commits       string
	Template      string
}

type GenerateOutput struct {
	Title  string
	Body   string
	Labels []string
}

// generatePRContent uses the input data to generate PR content
func generatePRContent(input GenerateInput) (GenerateOutput, error) {
	output := GenerateOutput{}

	// Extract type of change from branch name (feature, fix, etc.)
	branchType := extractBranchType(input.Branch)

	// Generate title based on branch name and commits
	output.Title = generateTitle(input.Branch, input.Commits)

	// Generate body
	if input.Template != "" {
		// If we have a template, fill it out
		output.Body = fillTemplate(input.Template, input)
	} else {
		// Otherwise generate a standard format
		output.Body = generateBody(input)
	}

	// Suggest labels based on changes
	output.Labels = suggestLabels(branchType, input.Diff)

	return output, nil
}

func extractBranchType(branch string) string {
	branch = strings.ToLower(branch)
	if strings.HasPrefix(branch, "feature/") || strings.HasPrefix(branch, "feat/") {
		return "feature"
	}
	if strings.HasPrefix(branch, "fix/") || strings.HasPrefix(branch, "bugfix/") {
		return "bug"
	}
	if strings.HasPrefix(branch, "docs/") {
		return "documentation"
	}
	if strings.HasPrefix(branch, "chore/") {
		return "maintenance"
	}
	return "enhancement"
}

func generateTitle(branch string, commits string) string {
	// Extract type from branch name
	branchType := extractBranchType(branch)
	conventionalType := convertToConventionalType(branchType)

	// Get the scope from the branch name if it exists
	scope := extractScope(branch)

	// Get a description from the branch name or first commit
	description := extractDescription(branch, commits)

	// Format as conventional commit
	if scope != "" {
		return fmt.Sprintf("%s(%s): %s", conventionalType, scope, description)
	}
	return fmt.Sprintf("%s: %s", conventionalType, description)
}

func convertToConventionalType(branchType string) string {
	switch branchType {
	case "feature":
		return "feat"
	case "bug":
		return "fix"
	case "documentation":
		return "docs"
	case "maintenance":
		return "chore"
	case "enhancement":
		return "feat"
	default:
		return "chore"
	}
}

func extractScope(branch string) string {
	// Look for patterns like feature/api/... or feat/ui/...
	parts := strings.Split(branch, "/")
	if len(parts) >= 3 {
		return parts[1]
	}
	return ""
}

func extractDescription(branch string, commits string) string {
	// First try to get a meaningful description from the branch name
	name := branch
	if idx := strings.LastIndex(branch, "/"); idx != -1 {
		name = branch[idx+1:]
	}

	// Clean up the branch name
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ToLower(name)

	// If the branch name is not descriptive enough, use the first commit message
	if len(name) < 10 {
		commitLines := strings.Split(commits, "\n")
		if len(commitLines) > 0 && commitLines[0] != "" {
			// If the commit message is already in conventional format, extract just the description
			commit := commitLines[0]
			if colonIdx := strings.Index(commit, ": "); colonIdx != -1 {
				return strings.TrimSpace(commit[colonIdx+2:])
			}
			return strings.TrimSpace(commit)
		}
	}

	return name
}

func fillTemplate(template string, input GenerateInput) string {
	// Split template into sections
	sections := strings.Split(template, "\n## ")

	var result strings.Builder

	for _, section := range sections {
		if strings.TrimSpace(section) == "" {
			continue
		}

		// Keep section headers
		if !strings.HasPrefix(section, "## ") {
			result.WriteString("## ")
		}

		lines := strings.Split(section, "\n")
		sectionTitle := strings.TrimSpace(lines[0])

		// Fill out each section based on its title
		switch strings.ToLower(sectionTitle) {
		case "description", "what does this pr do?", "summary":
			result.WriteString(section + "\n")
			result.WriteString(generateSummary(input))
		case "changes", "what changed?":
			result.WriteString(section + "\n")
			result.WriteString(generateChanges(input))
		case "testing", "how has this been tested?":
			result.WriteString(section + "\n")
			result.WriteString("This PR has been tested locally with the following checks:\n")
			result.WriteString("- [ ] Unit tests\n")
			result.WriteString("- [ ] Integration tests\n")
			result.WriteString("- [ ] Manual testing\n")
		default:
			// Keep other sections as is
			result.WriteString(section + "\n")
		}
	}

	return result.String()
}

func generateBody(input GenerateInput) string {
	var body strings.Builder

	body.WriteString("## Description\n\n")
	body.WriteString(generateSummary(input))

	body.WriteString("\n## Changes\n\n")
	body.WriteString(generateChanges(input))

	body.WriteString("\n## Testing\n\n")
	body.WriteString("This PR has been tested locally with the following checks:\n")
	body.WriteString("- [ ] Unit tests\n")
	body.WriteString("- [ ] Integration tests\n")
	body.WriteString("- [ ] Manual testing\n")

	return body.String()
}

func generateSummary(input GenerateInput) string {
	// Extract the first commit message as it usually contains the main change
	commitLines := strings.Split(input.Commits, "\n")
	if len(commitLines) > 0 {
		return commitLines[0]
	}
	return "Updates and improvements"
}

func generateChanges(input GenerateInput) string {
	var changes strings.Builder

	// Add commit history
	commits := strings.Split(input.Commits, "\n")
	for _, commit := range commits {
		if strings.TrimSpace(commit) != "" {
			changes.WriteString("- " + commit + "\n")
		}
	}

	return changes.String()
}

func suggestLabels(branchType string, diff string) []string {
	labels := []string{branchType}

	// Add more labels based on the changes
	if strings.Contains(diff, "test") {
		labels = append(labels, "tests")
	}
	if strings.Contains(diff, "doc") {
		labels = append(labels, "documentation")
	}

	return labels
}
