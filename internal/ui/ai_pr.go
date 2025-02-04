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

	// Get the diff
	diff, err := g.GetDiff()
	if err != nil {
		return form, fmt.Errorf("failed to get diff: %w", err)
	}

	// Get commit messages
	commits, err := g.Log(branch, 10, false, false)
	if err != nil {
		return form, fmt.Errorf("failed to get commit history: %w", err)
	}

	// Try to get PR template
	template, _ := ghc.GetPRTemplate()

	// Generate PR content using the collected information
	content, err := generatePRContent(GenerateInput{
		Branch:        branch,
		DefaultBranch: defaultBranch,
		Diff:          diff,
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
	// Remove prefix from branch name
	name := branch
	if idx := strings.Index(branch, "/"); idx != -1 {
		name = branch[idx+1:]
	}

	// Convert to title case and replace dashes/underscores with spaces
	title := strings.ReplaceAll(name, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")
	title = strings.Title(title)

	return title
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
