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

	// Get all changes: both staged and unstaged
	var allDiff strings.Builder

	// First get staged changes
	stagedDiff, err := g.GetDiff()
	if err == nil && stagedDiff != "" {
		allDiff.WriteString("Staged changes:\n")
		allDiff.WriteString(stagedDiff)
		allDiff.WriteString("\n")
	}

	// Then get unstaged changes
	if err := g.RunInteractive("add", "--intent-to-add", "."); err == nil {
		unstagedDiff, err := g.GetDiff()
		if err == nil && unstagedDiff != "" {
			allDiff.WriteString("\nUnstaged changes:\n")
			allDiff.WriteString(unstagedDiff)
		}
		// Restore the state
		g.RunInteractive("restore", "--staged", ".")
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
		Diff:          allDiff.String(),
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

	// Extract type of change from branch name and commits
	branchType := extractBranchType(input.Branch)
	commitTypes := extractCommitTypes(input.Commits)

	// Determine the most appropriate type based on both branch and commits
	finalType := determinePRType(branchType, commitTypes)

	// Generate title based on the determined type and changes
	output.Title = generateTitle(input.Branch, input.Commits)

	// Generate body
	if input.Template != "" {
		// If we have a template, fill it out
		output.Body = fillTemplate(input.Template, input)
	} else {
		// Otherwise generate a comprehensive format
		output.Body = generateComprehensiveBody(input)
	}

	// Suggest labels based on all available information
	output.Labels = suggestLabels(finalType, input.Diff, input.Commits)

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

func extractCommitTypes(commits string) []string {
	var types []string
	lines := strings.Split(commits, "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Extract conventional commit type
		if idx := strings.Index(line, ":"); idx > 0 {
			commitType := strings.TrimSpace(line[:idx])
			// Handle scoped commits like feat(ui)
			if scopeStart := strings.Index(commitType, "("); scopeStart > 0 {
				commitType = commitType[:scopeStart]
			}
			types = append(types, commitType)
		}
	}

	return types
}

func determinePRType(branchType string, commitTypes []string) string {
	// Count occurrences of each type in commits
	typeCounts := make(map[string]int)
	for _, t := range commitTypes {
		typeCounts[t]++
	}

	// If branch type is explicit, prefer it
	if branchType != "enhancement" {
		return branchType
	}

	// Otherwise use most common commit type
	maxCount := 0
	mostCommonType := "enhancement"
	for t, count := range typeCounts {
		if count > maxCount {
			maxCount = count
			mostCommonType = t
		}
	}

	return convertCommitTypeToLabel(mostCommonType)
}

func convertCommitTypeToLabel(commitType string) string {
	switch commitType {
	case "feat":
		return "feature"
	case "fix":
		return "bug"
	case "docs":
		return "documentation"
	case "chore":
		return "maintenance"
	case "refactor":
		return "refactor"
	case "test":
		return "testing"
	default:
		return "enhancement"
	}
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
	// First clean up the template by removing GitHub-flavored markdown comments
	template = cleanGitHubMarkdown(template)

	// Split template into sections, trying different section markers
	sections := splitIntoSections(template)

	var result strings.Builder
	for i, section := range sections {
		if strings.TrimSpace(section) == "" {
			continue
		}

		// Keep section headers for all but first section
		if i > 0 {
			result.WriteString("\n## ")
		}

		lines := strings.Split(section, "\n")
		if len(lines) == 0 {
			continue
		}

		sectionTitle := extractSectionTitle(lines[0])
		if sectionTitle == "" {
			// If no title found, keep the section as is
			result.WriteString(section)
			continue
		}

		// Get the template content for this section (excluding the title)
		templateContent := strings.Join(lines[1:], "\n")

		// Fill out each section based on its title
		switch strings.ToLower(sectionTitle) {
		case "description", "what does this pr do?", "summary", "overview":
			result.WriteString(sectionTitle + "\n")
			result.WriteString(fillSectionContent(templateContent, generateSummary(input)))
		case "changes", "what changed?", "implementation details":
			result.WriteString(sectionTitle + "\n")
			result.WriteString(fillSectionContent(templateContent, generateChanges(input)))
		case "testing", "how has this been tested?", "test plan":
			result.WriteString(sectionTitle + "\n")
			result.WriteString(fillSectionContent(templateContent,
				"This PR has been tested locally with the following checks:\n"+
					"- [ ] Unit tests\n"+
					"- [ ] Integration tests\n"+
					"- [ ] Manual testing\n"))
		case "breaking changes", "breaking":
			result.WriteString(sectionTitle + "\n")
			content := "No breaking changes.\n"
			if hasBreakingChanges(input) {
				content = "This PR contains breaking changes:\n" + generateBreakingChanges(input)
			}
			result.WriteString(fillSectionContent(templateContent, content))
		default:
			// Keep other sections as is, preserving their original format
			result.WriteString(section)
		}
		result.WriteString("\n")
	}

	return strings.TrimSpace(result.String())
}

// cleanGitHubMarkdown removes GitHub-flavored markdown comments and cleans up the template
func cleanGitHubMarkdown(template string) string {
	lines := strings.Split(template, "\n")
	var cleaned []string
	inComment := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines at the start
		if len(cleaned) == 0 && trimmed == "" {
			continue
		}

		// Handle comment blocks
		if strings.HasPrefix(trimmed, "<!--") {
			inComment = true
			if strings.HasSuffix(trimmed, "-->") {
				inComment = false
			}
			continue
		}
		if strings.HasSuffix(trimmed, "-->") {
			inComment = false
			continue
		}
		if inComment {
			continue
		}

		// Keep the line if it's not a comment
		cleaned = append(cleaned, line)
	}

	return strings.Join(cleaned, "\n")
}

// splitIntoSections splits the template into sections based on various header markers
func splitIntoSections(template string) []string {
	// Try different section markers in order of preference
	markers := []string{"\n## ", "\n# ", "\r\n## ", "\r\n# "}

	var sections []string
	for _, marker := range markers {
		sections = strings.Split(template, marker)
		if len(sections) > 1 {
			// Found a valid marker, process the sections
			for i := 1; i < len(sections); i++ {
				sections[i] = "## " + sections[i]
			}
			return sections
		}
	}

	// If no sections found, return the whole template as one section
	return []string{template}
}

// extractSectionTitle cleans up and extracts the actual title from a section header
func extractSectionTitle(header string) string {
	// Remove all heading markers
	title := strings.TrimLeft(header, "#")
	title = strings.TrimSpace(title)

	// Remove any trailing colons
	title = strings.TrimSuffix(title, ":")

	// Remove any markdown formatting
	title = strings.TrimSpace(strings.ReplaceAll(title, "`", ""))
	title = strings.TrimSpace(strings.ReplaceAll(title, "*", ""))
	title = strings.TrimSpace(strings.ReplaceAll(title, "_", ""))

	return title
}

// fillSectionContent handles template placeholders and formatting in section content
func fillSectionContent(template string, content string) string {
	if strings.TrimSpace(template) == "" {
		return content
	}

	// Look for common placeholder patterns
	placeholders := []string{
		"<!-- Write your description here -->",
		"<!-- Please include a summary of the changes -->",
		"<!-- Add your changes here -->",
		"<!-- List your changes here -->",
		"<!-- Describe your changes -->",
	}

	result := template
	for _, placeholder := range placeholders {
		if strings.Contains(result, placeholder) {
			result = strings.Replace(result, placeholder, content, 1)
			return result
		}
	}

	// If no placeholders found, append the content after any instructions
	if strings.Contains(strings.ToLower(template), "please") ||
		strings.Contains(strings.ToLower(template), "describe") ||
		strings.Contains(strings.ToLower(template), "list") {
		return template + "\n\n" + content
	}

	// If the template has checkboxes or bullet points, add content after them
	if strings.Contains(template, "- [ ]") || strings.Contains(template, "* [ ]") {
		return template + "\n" + content
	}

	// Otherwise, replace the template with our content
	return content
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

func suggestLabels(branchType string, diff string, commits string) []string {
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

func hasBreakingChanges(input GenerateInput) bool {
	commits := strings.Split(input.Commits, "\n")
	for _, commit := range commits {
		if strings.Contains(commit, "!:") || strings.Contains(strings.ToLower(commit), "breaking change") {
			return true
		}
	}
	return false
}

func generateBreakingChanges(input GenerateInput) string {
	var changes strings.Builder
	commits := strings.Split(input.Commits, "\n")
	for _, commit := range commits {
		if strings.Contains(commit, "!:") || strings.Contains(strings.ToLower(commit), "breaking change") {
			changes.WriteString("- " + commit + "\n")
		}
	}
	return changes.String()
}

func generateComprehensiveBody(input GenerateInput) string {
	var body strings.Builder

	// Description section
	body.WriteString("## Description\n\n")
	summary := generateSummary(input)
	if summary != "" {
		body.WriteString(summary + "\n\n")
	}

	// Changes section with better organization
	body.WriteString("## Changes\n\n")

	// Group changes by type
	changes := generateDetailedChanges(input)
	if changes != "" {
		body.WriteString(changes + "\n")
	}

	// Testing section with context-aware checklist
	body.WriteString("## Testing\n\n")
	body.WriteString(generateTestingSection(input))

	// Breaking changes section if relevant
	if hasBreakingChanges(input) {
		body.WriteString("\n## Breaking Changes\n\n")
		body.WriteString(generateBreakingChanges(input))
	}

	// Additional sections based on content
	if hasDocChanges(input.Diff) {
		body.WriteString("\n## Documentation\n\n")
		body.WriteString("- Documentation has been updated to reflect the changes\n")
	}

	if hasDependencyChanges(input.Diff) {
		body.WriteString("\n## Dependencies\n\n")
		body.WriteString("- Dependencies have been updated. Please review the changes carefully.\n")
	}

	return body.String()
}

func generateDetailedChanges(input GenerateInput) string {
	var changes strings.Builder
	commits := strings.Split(input.Commits, "\n")

	// Group commits by type
	typeGroups := make(map[string][]string)
	for _, commit := range commits {
		if commit == "" {
			continue
		}

		commitType := "Other"
		if idx := strings.Index(commit, ":"); idx > 0 {
			commitType = strings.TrimSpace(commit[:idx])
			// Handle scoped commits
			if scopeStart := strings.Index(commitType, "("); scopeStart > 0 {
				commitType = commitType[:scopeStart]
			}
		}

		typeGroups[commitType] = append(typeGroups[commitType], commit)
	}

	// Order types for consistent output
	typeOrder := []string{"feat", "fix", "refactor", "docs", "test", "chore", "Other"}

	for _, t := range typeOrder {
		if commits, ok := typeGroups[t]; ok && len(commits) > 0 {
			changes.WriteString(fmt.Sprintf("### %s\n\n", convertTypeToTitle(t)))
			for _, c := range commits {
				// Clean up commit message
				msg := c
				if idx := strings.Index(msg, ":"); idx > 0 {
					msg = strings.TrimSpace(msg[idx+1:])
				}
				changes.WriteString(fmt.Sprintf("- %s\n", msg))
			}
			changes.WriteString("\n")
		}
	}

	return changes.String()
}

func convertTypeToTitle(t string) string {
	switch t {
	case "feat":
		return "Features"
	case "fix":
		return "Bug Fixes"
	case "docs":
		return "Documentation Changes"
	case "chore":
		return "Maintenance"
	case "refactor":
		return "Code Refactoring"
	case "test":
		return "Tests"
	default:
		return "Other Changes"
	}
}

func generateTestingSection(input GenerateInput) string {
	var testing strings.Builder

	testing.WriteString("This PR has been tested with the following checks:\n\n")

	// Add relevant checkboxes based on changes
	if hasCodeChanges(input.Diff) {
		testing.WriteString("### Code Changes\n")
		testing.WriteString("- [ ] Unit tests have been added/updated\n")
		testing.WriteString("- [ ] Integration tests have been added/updated\n")
		testing.WriteString("- [ ] Manual testing has been performed\n\n")
	}

	if hasUIChanges(input.Diff) {
		testing.WriteString("### UI Changes\n")
		testing.WriteString("- [ ] Visual changes have been reviewed\n")
		testing.WriteString("- [ ] Cross-browser testing performed\n")
		testing.WriteString("- [ ] Responsive design verified\n\n")
	}

	if hasAPIChanges(input.Diff) {
		testing.WriteString("### API Changes\n")
		testing.WriteString("- [ ] API documentation updated\n")
		testing.WriteString("- [ ] API tests added/updated\n")
		testing.WriteString("- [ ] Backward compatibility verified\n\n")
	}

	return testing.String()
}

// Helper functions to detect types of changes
func hasCodeChanges(diff string) bool {
	return strings.Contains(diff, ".go") ||
		strings.Contains(diff, ".js") ||
		strings.Contains(diff, ".ts") ||
		strings.Contains(diff, ".py") ||
		strings.Contains(diff, ".java")
}

func hasUIChanges(diff string) bool {
	return strings.Contains(diff, ".css") ||
		strings.Contains(diff, ".scss") ||
		strings.Contains(diff, ".html") ||
		strings.Contains(diff, ".jsx") ||
		strings.Contains(diff, ".tsx")
}

func hasAPIChanges(diff string) bool {
	return strings.Contains(diff, "/api/") ||
		strings.Contains(diff, "openapi.yaml") ||
		strings.Contains(diff, "swagger.yaml") ||
		strings.Contains(diff, "proto")
}

func hasDocChanges(diff string) bool {
	return strings.Contains(diff, ".md") ||
		strings.Contains(diff, "docs/") ||
		strings.Contains(diff, "README") ||
		strings.Contains(diff, "CHANGELOG")
}

func hasDependencyChanges(diff string) bool {
	return strings.Contains(diff, "go.mod") ||
		strings.Contains(diff, "go.sum") ||
		strings.Contains(diff, "package.json") ||
		strings.Contains(diff, "requirements.txt") ||
		strings.Contains(diff, "Gemfile")
}
