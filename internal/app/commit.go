package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/sage/internal/ai"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
)

type CommitOptions struct {
	Message         string
	UseConventional bool
	AllowEmpty      bool
	PushAfterCommit bool
	UseAI           bool
	AutoAcceptAI    bool
	SuggestType     string // Type to suggest to AI
	ChangeType      string // Type to change to without regenerating
}

type CommitResult struct {
	ActualMessage string
	Pushed        bool
}

// changeCommitType changes the type of a conventional commit message
func changeCommitType(msg, newType string) string {
	if !strings.Contains(msg, ": ") {
		return fmt.Sprintf("%s: %s", newType, msg)
	}

	parts := strings.SplitN(msg, ": ", 2)
	oldType := parts[0]

	// Handle scoped commits (e.g., feat(api): message)
	if strings.Contains(oldType, "(") {
		typeParts := strings.SplitN(oldType, "(", 2)
		scope := "(" + typeParts[1] // includes the closing parenthesis
		return fmt.Sprintf("%s%s: %s", newType, scope, parts[1])
	}

	return fmt.Sprintf("%s: %s", newType, parts[1])
}

func Commit(g git.Service, opts CommitOptions) (*CommitResult, error) {
	isRepo, err := g.IsRepo()
	if err != nil {
		return nil, err
	}
	if !isRepo {
		return nil, fmt.Errorf("not a git repository")
	}

	// Check if .sage/ is already staged
	sageStaged, err := g.IsPathStaged(".sage/")
	if err != nil {
		return nil, err
	}

	// Stage all files, excluding .sage/ if it's not already staged
	if sageStaged {
		err = g.StageAll()
	} else {
		err = g.StageAllExcept([]string{".sage/"})
	}
	if err != nil {
		return nil, err
	}

	if opts.UseAI && !opts.AllowEmpty {
		// Stage all files first
		if err := g.StageAll(); err != nil {
			return nil, fmt.Errorf("failed to stage files: %w", err)
		}

		diff, err := g.GetDiff()
		if err != nil {
			return nil, fmt.Errorf("failed to get diff for AI commit message: %w", err)
		}
		if diff == "" {
			return nil, fmt.Errorf("no changes to commit; use --empty to allow empty")
		}
		client := ai.NewClient("")
		for {
			var msg string
			var err error

			if opts.SuggestType != "" {
				msg, err = client.GenerateCommitMessage(diff + "\n\nPlease use the commit type: " + opts.SuggestType)
			} else {
				msg, err = client.GenerateCommitMessage(diff)
			}

			if err != nil {
				return nil, fmt.Errorf("failed to generate AI commit message: %w", err)
			}

			// Ensure the message is in conventional commit format
			if !strings.Contains(msg, ":") {
				msg = "chore: " + msg
			}

			if opts.AutoAcceptAI {
				opts.Message = msg
				break
			}

			fmt.Printf("Generated commit message: %q\n", msg)
			confirm := ""
			err = survey.AskOne(&survey.Select{
				Message: "What would you like to do?",
				Options: []string{
					"Accept",
					"Regenerate",
					"Change type",
					"Enter manually",
				},
			}, &confirm)
			if err != nil {
				return nil, err
			}

			switch confirm {
			case "Accept":
				opts.Message = msg
				break
			case "Change type":
				newType := ""
				err = survey.AskOne(&survey.Select{
					Message: "Select new commit type:",
					Options: []string{
						"feat", "fix", "docs", "style",
						"refactor", "test", "chore",
					},
				}, &newType)
				if err != nil {
					return nil, err
				}
				opts.Message = changeCommitType(msg, newType)
				break
			case "Enter manually":
				opts.UseAI = false
				opts.Message = ""
				break
				// For "Regenerate", continue the loop
			}

			if opts.Message != "" || !opts.UseAI {
				break
			}
		}
	} else if opts.Message == "" {
		// prompt
		msg, scope, ctype, err := ui.AskCommitMessage(opts.UseConventional)
		if err != nil {
			return nil, err
		}
		if opts.UseConventional {
			if scope != "" {
				opts.Message = fmt.Sprintf("%s(%s): %s", ctype, scope, msg)
			} else {
				opts.Message = fmt.Sprintf("%s: %s", ctype, msg)
			}
		} else {
			opts.Message = msg
		}
	} else if opts.UseConventional && !strings.Contains(opts.Message, ":") {
		opts.Message = "chore: " + opts.Message
	}

	// Handle type change if requested
	if opts.ChangeType != "" {
		opts.Message = changeCommitType(opts.Message, opts.ChangeType)
	}

	if !opts.AllowEmpty {
		clean, err := g.IsClean()
		if err != nil {
			return nil, err
		}
		if clean {
			return nil, fmt.Errorf("no changes to commit; use --empty to allow empty")
		}
		if err := g.StageAll(); err != nil {
			return nil, err
		}
	}

	if err := g.Commit(opts.Message, opts.AllowEmpty); err != nil {
		return nil, err
	}
	res := &CommitResult{ActualMessage: opts.Message}

	if opts.PushAfterCommit {
		branch, err := g.CurrentBranch()
		if err != nil {
			return nil, err
		}
		if err := g.Push(branch, false); err != nil {
			return nil, err
		}
		res.Pushed = true
	}

	return res, nil
}

// CommitMultiple creates multiple commits based on AI-grouped changes
func CommitMultiple(g git.Service, opts CommitOptions) error {
	// Always use conventional commits in multiple mode
	opts.UseConventional = true

	// First, get all changes and group them using the same logic as stage --ai
	status, err := g.StatusPorcelain()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	// Parse status into FileStatus structs (reusing from stage.go)
	var files []FileStatus
	for _, line := range strings.Split(strings.TrimSpace(status), "\n") {
		if line == "" {
			continue
		}

		statusCode := line[:2]
		path := strings.TrimSpace(line[3:])

		if strings.Contains(path, " -> ") {
			parts := strings.Split(path, " -> ")
			path = parts[1]
		}

		if statusCode[0] == ' ' || statusCode[0] == '?' {
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
		return fmt.Errorf("no changes to commit")
	}

	// Initialize AI client
	client := ai.NewClient("")
	if client.APIKey == "" {
		return fmt.Errorf("AI features require an OpenAI API key")
	}

	// Build diff for AI analysis
	var diffBuilder strings.Builder
	diffBuilder.WriteString("Files to analyze:\n")
	for _, file := range files {
		diffBuilder.WriteString(fmt.Sprintf("- %s (%s)\n", file.Path, file.Status))
	}
	diffBuilder.WriteString("\nChanges:\n")

	for _, file := range files {
		if file.Status == "Added" {
			content, err := os.ReadFile(file.Path)
			if err == nil {
				diffBuilder.WriteString(fmt.Sprintf("\nNew file: %s\n", file.Path))
				diffBuilder.WriteString("```\n")
				diffBuilder.WriteString(string(content))
				diffBuilder.WriteString("\n```\n")
			}
		} else {
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

	// Get AI to group the changes
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
		return fmt.Errorf("failed to analyze changes: %w", err)
	}

	// Parse the groups
	groups := parseGroups(response)
	if len(groups) == 0 {
		return fmt.Errorf("failed to group changes")
	}

	// Classify files into groups
	for _, file := range files {
		prompt := fmt.Sprintf(`Based on the file path and its changes, which group does this file belong to?

File: %s
Status: %s

Available groups:
%s

Return only the exact name of the most appropriate group from the list above.`, file.Path, file.Status, formatGroups(groups))

		groupName, err := client.GenerateCommitMessage(prompt)
		if err != nil {
			groupName = groups[0].Name
		}

		groupName = strings.TrimSpace(groupName)
		found := false
		for i := range groups {
			if strings.EqualFold(groups[i].Name, groupName) {
				groups[i].Files = append(groups[i].Files, file)
				found = true
				break
			}
		}

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
		return fmt.Errorf("no valid groups found")
	}

	// Create options for the group selector
	var options []string
	for _, group := range nonEmptyGroups {
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

		groupOption := fmt.Sprintf("%s: %s\n%s",
			strings.ToUpper(group.Name),
			group.Description,
			fileList.String())
		options = append(options, groupOption)
	}

	// Show interactive group selector
	var selected []string
	groupPrompt := &survey.MultiSelect{
		Message:  "Select groups to commit:",
		Options:  options,
		PageSize: 15,
	}
	err = survey.AskOne(groupPrompt, &selected)
	if err != nil {
		return fmt.Errorf("selection cancelled: %w", err)
	}

	if len(selected) == 0 {
		return fmt.Errorf("no groups selected")
	}

	// Create a commit for each selected group
	for _, sel := range selected {
		groupName := strings.ToLower(strings.Split(sel, ":")[0])
		var selectedGroup *ChangeGroup
		for i := range nonEmptyGroups {
			if strings.EqualFold(nonEmptyGroups[i].Name, groupName) {
				selectedGroup = &nonEmptyGroups[i]
				break
			}
		}

		if selectedGroup == nil {
			continue
		}

		// Stage only the files for this group
		for _, file := range selectedGroup.Files {
			if err := g.RunInteractive("add", file.Path); err != nil {
				return fmt.Errorf("failed to stage %s: %w", file.Path, err)
			}
		}

		// Get AI to generate a commit message for this group
		var groupDiff strings.Builder
		groupDiff.WriteString(fmt.Sprintf("Group: %s\nDescription: %s\n\nFiles changed:\n",
			selectedGroup.Name, selectedGroup.Description))
		for _, file := range selectedGroup.Files {
			groupDiff.WriteString(fmt.Sprintf("- %s (%s)\n", file.Path, file.Status))
		}
		diff, err := g.GetDiff()
		if err == nil {
			groupDiff.WriteString("\nChanges:\n")
			groupDiff.WriteString(diff)
		}

		// Add type suggestion based on group name
		var typeHint string
		switch {
		case strings.Contains(strings.ToLower(selectedGroup.Name), "feat"):
			typeHint = "feat"
		case strings.Contains(strings.ToLower(selectedGroup.Name), "fix"):
			typeHint = "fix"
		case strings.Contains(strings.ToLower(selectedGroup.Name), "doc"):
			typeHint = "docs"
		case strings.Contains(strings.ToLower(selectedGroup.Name), "test"):
			typeHint = "test"
		case strings.Contains(strings.ToLower(selectedGroup.Name), "refactor"):
			typeHint = "refactor"
		case strings.Contains(strings.ToLower(selectedGroup.Name), "style"):
			typeHint = "style"
		default:
			typeHint = "chore"
		}

		prompt := groupDiff.String()
		if typeHint != "" {
			prompt += fmt.Sprintf("\nPlease use the commit type: %s", typeHint)
		}

		commitMsg, err := client.GenerateCommitMessage(prompt)
		if err != nil {
			return fmt.Errorf("failed to generate commit message: %w", err)
		}

		// Ensure conventional commit format
		if !strings.Contains(commitMsg, ":") {
			commitMsg = fmt.Sprintf("%s: %s", typeHint, commitMsg)
		}

		if !opts.AutoAcceptAI {
			fmt.Printf("\nFor group %q:\n", strings.ToUpper(selectedGroup.Name))
			fmt.Printf("Generated commit message: %q\n", commitMsg)
			confirm := ""
			err = survey.AskOne(&survey.Select{
				Message: "What would you like to do?",
				Options: []string{
					"Accept",
					"Regenerate",
					"Change type",
					"Enter manually",
					"Skip group",
				},
			}, &confirm)
			if err != nil {
				return err
			}

			switch confirm {
			case "Regenerate":
				// Try again without the type hint
				commitMsg, err = client.GenerateCommitMessage(groupDiff.String())
				if err != nil {
					return fmt.Errorf("failed to regenerate commit message: %w", err)
				}
				if !strings.Contains(commitMsg, ":") {
					commitMsg = fmt.Sprintf("%s: %s", typeHint, commitMsg)
				}
			case "Change type":
				newType := ""
				err = survey.AskOne(&survey.Select{
					Message: "Select new commit type:",
					Options: []string{
						"feat", "fix", "docs", "style",
						"refactor", "test", "chore",
					},
				}, &newType)
				if err != nil {
					return err
				}
				commitMsg = changeCommitType(commitMsg, newType)
			case "Enter manually":
				msg, scope, ctype, err := ui.AskCommitMessage(true)
				if err != nil {
					return err
				}
				if scope != "" {
					commitMsg = fmt.Sprintf("%s(%s): %s", ctype, scope, msg)
				} else {
					commitMsg = fmt.Sprintf("%s: %s", ctype, msg)
				}
			case "Skip group":
				// Unstage files and continue to next group
				for _, file := range selectedGroup.Files {
					g.RunInteractive("restore", "--staged", file.Path)
				}
				continue
			}
		}

		// Create the commit
		if err := g.Commit(commitMsg, opts.AllowEmpty); err != nil {
			return fmt.Errorf("failed to create commit: %w", err)
		}

		fmt.Printf("%s Created commit: %s\n", ui.Green("✓"), commitMsg)
	}

	// Push if requested
	if opts.PushAfterCommit {
		branch, err := g.CurrentBranch()
		if err != nil {
			return err
		}
		if err := g.Push(branch, false); err != nil {
			return err
		}
		fmt.Printf("%s Pushed changes to %s\n", ui.Green("✓"), branch)
	}

	return nil
}
