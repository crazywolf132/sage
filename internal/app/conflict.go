package app

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
)

// ConflictOptions contains options for handling conflicts
type ConflictOptions struct {
	AutoResolve bool   // Automatically resolve conflicts where possible
	Editor      string // Editor to use for conflict resolution
}

// ResolveConflicts helps manage and resolve conflicts
func ResolveConflicts(g git.Service, opts ConflictOptions) error {
	sg, ok := g.(*git.ShellGit)
	if !ok {
		return fmt.Errorf("invalid git service for conflict resolution")
	}

	// Get list of conflicted files
	conflictsOutput, err := sg.ListConflictedFiles()
	if err != nil {
		return fmt.Errorf("failed to list conflicts: %w", err)
	}

	conflicts := strings.Split(strings.TrimSpace(conflictsOutput), "\n")
	if len(conflicts) == 0 || (len(conflicts) == 1 && conflicts[0] == "") {
		return fmt.Errorf("no conflicts detected")
	}

	ui.Success(fmt.Sprintf("Found %d files with conflicts", len(conflicts)))

	// Display conflicts with status indicators
	for i, file := range conflicts {
		if file == "" {
			continue
		}
		ui.Info(fmt.Sprintf("%d. %s", i+1, file))
	}
	fmt.Println()

	// Ask which files to edit
	var selectedFiles []string
	prompt := &survey.MultiSelect{
		Message: "Select files to edit:",
		Options: conflicts,
	}
	survey.AskOne(prompt, &selectedFiles)

	if len(selectedFiles) == 0 {
		ui.Warning("No files selected. You'll need to resolve conflicts manually.")
		return nil
	}

	// Open selected files in editor
	editor := getEditor(opts.Editor)
	if editor == "" {
		ui.Warning("No editor configured. Please resolve conflicts manually.")
		return nil
	}

	for _, file := range selectedFiles {
		if err := openInEditor(editor, file); err != nil {
			ui.Warning(fmt.Sprintf("Failed to open %s: %v", file, err))
		}
	}

	ui.Info("\nAfter resolving conflicts:")
	ui.Info("1. Save and close the files")
	ui.Info("2. Use 'git add' to mark conflicts as resolved")
	ui.Info("3. Run 'sage sync --continue' to complete the operation")

	return nil
}

// getEditor returns the editor to use based on user preferences
func getEditor(customEditor string) string {
	if customEditor != "" {
		return customEditor
	}

	// Check if EDITOR env var is set
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	// Check common editors based on OS
	switch runtime.GOOS {
	case "darwin":
		for _, editor := range []string{"code", "subl", "atom", "vim", "nano"} {
			if _, err := exec.LookPath(editor); err == nil {
				return editor
			}
		}
		// Default to TextEdit on macOS as last resort
		return "open -a TextEdit"
	case "linux":
		for _, editor := range []string{"code", "subl", "gedit", "vim", "nano"} {
			if _, err := exec.LookPath(editor); err == nil {
				return editor
			}
		}
	case "windows":
		for _, editor := range []string{"code", "notepad++"} {
			if _, err := exec.LookPath(editor); err == nil {
				return editor
			}
		}
		// Default to notepad on Windows as last resort
		return "notepad"
	}

	return ""
}

// openInEditor opens the given file in the specified editor
func openInEditor(editor, file string) error {
	var cmd *exec.Cmd

	// Handle special cases
	if editor == "open -a TextEdit" {
		cmd = exec.Command("open", "-a", "TextEdit", file)
	} else {
		cmd = exec.Command(editor, file)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
