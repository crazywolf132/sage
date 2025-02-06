package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/sage/internal/ai"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
)

type FileStatus struct {
	Path   string
	Status string
}

type ChangeGroup struct {
	Name        string
	Description string
	Files       []FileStatus
}

func StageFiles(g git.Service, patterns []string, useAI bool) error {
	// Get the current status
	status, err := g.StatusPorcelain()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	// Parse status into FileStatus structs
	var files []FileStatus
	for _, line := range strings.Split(strings.TrimSpace(status), "\n") {
		if line == "" {
			continue
		}

		// Status format is XY PATH or XY PATH -> PATH2 for renames
		// X is status in staging area, Y is status in working tree
		statusCode := line[:2]
		path := strings.TrimSpace(line[3:])

		// Handle renamed files
		if strings.Contains(path, " -> ") {
			parts := strings.Split(path, " -> ")
			path = parts[1] // Use the new path
		}

		// Include files that are:
		// - Modified in working tree (M)
		// - Added/untracked (A or ?)
		// - Deleted (D)
		// - Renamed (R)
		// And not already staged (X is space or ?)
		if statusCode[0] == ' ' || statusCode[0] == '?' {
			// Get human-readable status
			var humanStatus string
			switch statusCode[1] {
			case 'M':
				humanStatus = "Modified"
			case 'A', '?':
				humanStatus = "Added"
			case 'D':
				humanStatus = "Deleted"
			case 'R':
				humanStatus = "Renamed"
			default:
				humanStatus = "Unknown"
			}

			files = append(files, FileStatus{
				Path:   path,
				Status: humanStatus,
			})
		}
	}

	if len(files) == 0 {
		fmt.Printf("%s No files to stage\n", ui.Yellow("!"))
		return nil
	}

	// If patterns are provided, stage matching files
	if len(patterns) > 0 {
		var stagedFiles []string
		for _, pattern := range patterns {
			for _, file := range files {
				matched, err := filepath.Match(pattern, file.Path)
				if err != nil {
					return fmt.Errorf("invalid pattern %q: %w", pattern, err)
				}
				if matched {
					if err := g.RunInteractive("add", file.Path); err != nil {
						return fmt.Errorf("failed to stage %s: %w", file.Path, err)
					}
					stagedFiles = append(stagedFiles, file.Path)
				}
			}
		}

		if len(stagedFiles) > 0 {
			fmt.Printf("%s Staged %d files matching patterns\n", ui.Green("✓"), len(stagedFiles))
		} else {
			fmt.Printf("%s No files matched the provided patterns\n", ui.Yellow("!"))
		}
		return nil
	}

	if useAI {
		// Initialize AI client
		client := ai.NewClient("")
		if client.APIKey == "" {
			fmt.Printf("%s No OpenAI API key found, falling back to manual selection\n", ui.Yellow("!"))
			return StageFiles(g, patterns, false)
		}

		// Get the diff for AI analysis
		var diffBuilder strings.Builder
		diffBuilder.WriteString("Files to analyze:\n")

		// First list all files
		for _, file := range files {
			diffBuilder.WriteString(fmt.Sprintf("- %s (%s)\n", file.Path, file.Status))
		}
		diffBuilder.WriteString("\nChanges:\n")

		// Then add content/diff for each file
		for _, file := range files {
			if file.Status == "Added" {
				// For untracked files, get their content
				content, err := os.ReadFile(file.Path)
				if err == nil {
					diffBuilder.WriteString(fmt.Sprintf("\nNew file: %s\n", file.Path))
					diffBuilder.WriteString("```\n")
					diffBuilder.WriteString(string(content))
					diffBuilder.WriteString("\n```\n")
				}
			} else {
				// For tracked files, get their diff
				if err := g.RunInteractive("add", "--intent-to-add", file.Path); err == nil {
					diff, err := g.GetDiff()
					if err == nil {
						diffBuilder.WriteString(fmt.Sprintf("\nModified file: %s\n", file.Path))
						diffBuilder.WriteString("```diff\n")
						diffBuilder.WriteString(diff)
						diffBuilder.WriteString("\n```\n")
					}
					g.RunInteractive("restore", "--staged", file.Path)
				}
			}
		}

		// Generate groups based on the changes
		prompt := fmt.Sprintf(`You are a helpful assistant that analyzes code changes and groups them by functionality.
Please analyze these changes and group them into logical categories.

For each group:
1. Give it a short, descriptive name (e.g., "auth", "ui", "docs")
2. Provide a brief description of what the changes in that group do

%s

Format your response exactly like this example:
feature: Add new user authentication system
docs: Update API documentation
test: Add integration tests for auth system

Your response:`, diffBuilder.String())

		response, err := client.GenerateCommitMessage(prompt)
		if err != nil {
			fmt.Printf("%s Failed to analyze changes with AI: %v\n", ui.Yellow("!"), err)
			return StageFiles(g, patterns, false)
		}

		// Parse the groups
		groups := parseGroups(response)
		if len(groups) == 0 {
			fmt.Printf("%s Failed to parse AI response, falling back to manual selection\n", ui.Yellow("!"))
			return StageFiles(g, patterns, false)
		}

		// For each file, ask AI which group it belongs to
		for _, file := range files {
			prompt := fmt.Sprintf(`Based on the file path and its changes, which group does this file belong to?

File: %s
Status: %s

Available groups:
%s

Return only the exact name of the most appropriate group from the list above.`, file.Path, file.Status, formatGroups(groups))

			groupName, err := client.GenerateCommitMessage(prompt)
			if err != nil {
				// If AI fails to classify, put in the first group
				groupName = groups[0].Name
			}

			// Add file to the appropriate group
			groupName = strings.TrimSpace(groupName)
			found := false
			for i := range groups {
				if strings.EqualFold(groups[i].Name, groupName) {
					groups[i].Files = append(groups[i].Files, file)
					found = true
					break
				}
			}

			// If no matching group found, add to first group
			if !found {
				groups[0].Files = append(groups[0].Files, file)
			}
		}

		// Filter out empty groups
		var nonEmptyGroups []ChangeGroup
		for _, group := range groups {
			if len(group.Files) > 0 {
				nonEmptyGroups = append(nonEmptyGroups, group)
			}
		}

		if len(nonEmptyGroups) == 0 {
			fmt.Printf("%s No valid groups found, falling back to manual selection\n", ui.Yellow("!"))
			return StageFiles(g, patterns, false)
		}

		// Create options for the group selector
		var options []string
		for _, group := range nonEmptyGroups {
			// Create a formatted file list with status indicators
			var fileList strings.Builder
			for _, file := range group.Files {
				var statusSymbol string
				switch file.Status {
				case "Added":
					statusSymbol = "+"
				case "Modified":
					statusSymbol = "~"
				case "Deleted":
					statusSymbol = "-"
				case "Renamed":
					statusSymbol = "→"
				default:
					statusSymbol = " "
				}
				fileList.WriteString(fmt.Sprintf("  %s %s\n", statusSymbol, file.Path))
			}

			// Format the group option with a clear header and indented file list
			groupOption := fmt.Sprintf("%s: %s\n%s",
				strings.ToUpper(group.Name),
				group.Description,
				fileList.String())
			options = append(options, groupOption)
		}

		// Show interactive group selector
		var selected []string
		groupPrompt := &survey.MultiSelect{
			Message:  "Select groups of changes to stage:",
			Options:  options,
			PageSize: 15,
		}
		err = survey.AskOne(groupPrompt, &selected)
		if err != nil {
			return fmt.Errorf("selection cancelled: %w", err)
		}

		if len(selected) == 0 {
			fmt.Printf("%s No groups selected to stage\n", ui.Yellow("!"))
			return nil
		}

		// Stage files from selected groups
		var stagedCount int
		for _, sel := range selected {
			// Extract group name from the uppercase format
			groupName := strings.ToLower(strings.Split(sel, ":")[0])
			for _, group := range nonEmptyGroups {
				if strings.EqualFold(group.Name, groupName) {
					for _, file := range group.Files {
						if err := g.RunInteractive("add", file.Path); err != nil {
							return fmt.Errorf("failed to stage %s: %w", file.Path, err)
						}
						stagedCount++
					}
					break
				}
			}
		}

		fmt.Printf("%s Staged %d files from %d groups\n", ui.Green("✓"), stagedCount, len(selected))
		return nil
	}

	// Create options for the interactive selector
	var options []string
	for _, file := range files {
		options = append(options, fmt.Sprintf("%s (%s)", file.Path, file.Status))
	}

	// Show interactive selector
	var selected []string
	prompt := &survey.MultiSelect{
		Message:  "Select files to stage:",
		Options:  options,
		PageSize: 15,
	}
	err = survey.AskOne(prompt, &selected)
	if err != nil {
		return fmt.Errorf("selection cancelled: %w", err)
	}

	if len(selected) == 0 {
		fmt.Printf("%s No files selected to stage\n", ui.Yellow("!"))
		return nil
	}

	// Stage selected files
	for _, sel := range selected {
		// Extract path from selection (remove status)
		path := strings.TrimSpace(strings.Split(sel, " (")[0])
		if err := g.RunInteractive("add", path); err != nil {
			return fmt.Errorf("failed to stage %s: %w", path, err)
		}
	}

	fmt.Printf("%s Staged %d files\n", ui.Green("✓"), len(selected))
	return nil
}

func formatFileList(files []FileStatus) string {
	var result []string
	for _, file := range files {
		result = append(result, fmt.Sprintf("%s (%s)", file.Path, file.Status))
	}
	return strings.Join(result, "\n")
}

func formatGroups(groups []ChangeGroup) string {
	var result []string
	for _, group := range groups {
		result = append(result, fmt.Sprintf("%s: %s", group.Name, group.Description))
	}
	return strings.Join(result, "\n")
}

func parseGroups(response string) []ChangeGroup {
	var groups []ChangeGroup
	lines := strings.Split(strings.TrimSpace(response), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		groups = append(groups, ChangeGroup{
			Name:        strings.TrimSpace(parts[0]),
			Description: strings.TrimSpace(parts[1]),
			Files:       []FileStatus{},
		})
	}
	return groups
}

func indent(text string, prefix string) string {
	var result []string
	for _, line := range strings.Split(text, "\n") {
		if line != "" {
			result = append(result, prefix+line)
		}
	}
	return strings.Join(result, "\n")
}
