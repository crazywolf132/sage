package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/charmbracelet/huh"
)

// PRForm represents the structure for pull request form data
type PRForm struct {
	Title string
	Body  string
}

// CommitForm represents the structure for commit form data
type CommitForm struct {
	Type    string
	Scope   string
	Message string
}

// GetCommitDetailsFunc is the function type for getting commit details
type GetCommitDetailsFunc func(useConventional bool) (CommitForm, error)

// GetCommitDetails is the function variable that can be reassigned for testing
var GetCommitDetails = getCommitDetails

// getCommitDetails prompts the user for commit details with conventional commit support
func getCommitDetails(useConventional bool) (CommitForm, error) {
	var form CommitForm

	if !useConventional {
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Commit Message").
					Value(&form.Message).
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("commit message cannot be empty")
						}
						return nil
					}),
			),
		).Run()
		return form, err
	}

	types := []string{"feat", "fix", "docs", "style", "refactor", "test", "chore"}

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Type").
				Options(huh.NewOptions(types...)...).
				Value(&form.Type).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("commit type must be selected")
					}
					return nil
				}),
			huh.NewInput().
				Title("Scope (optional)").
				Value(&form.Scope),
			huh.NewInput().
				Title("Message").
				Value(&form.Message).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("commit message cannot be empty")
					}
					return nil
				}),
		),
	).Run()

	return form, err
}

// getPreferredEditor returns the user's preferred editor from environment variables
func getPreferredEditor() string {
	if editor := os.Getenv("SAGE_EDITOR"); editor != "" {
		return editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	return "vi" // fallback to vi if no editor is specified
}

// editInVi opens a temporary file in the preferred editor and returns the edited contents
func editInVi(initialContent string) (string, error) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "sage-pr-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write initial content
	if _, err := tmpFile.WriteString(initialContent); err != nil {
		return "", fmt.Errorf("failed to write to temporary file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Get the preferred editor
	editor := getPreferredEditor()

	// Open editor
	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor '%s' returned error: %w", editor, err)
	}

	// Read the edited content
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read edited file: %w", err)
	}

	return string(content), nil
}

// GetPRDetails prompts the user for pull request details
func GetPRDetails(initialForm PRForm) (PRForm, error) {
	var form PRForm = initialForm
	var err error

	// First get the title using huh
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Title").
				Value(&form.Title).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("PR title cannot be empty")
					}
					return nil
				}),
		),
	).Run()

	if err != nil {
		return form, err
	}

	// Then edit the body using vi directly
	form.Body, err = editInVi(form.Body)
	if err != nil {
		return form, fmt.Errorf("failed to edit PR body: %w", err)
	}

	// Validate the body
	if len(form.Body) < 10 {
		return form, fmt.Errorf("PR description should be at least 10 characters")
	}

	return form, nil
}

// BackupPRForm saves the form data to a backup file
func BackupPRForm(form PRForm) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	backupDir := filepath.Join(homeDir, ".sage", "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}

	backupFile := filepath.Join(backupDir, "pr_form_backup.json")
	data, err := json.Marshal(form)
	if err != nil {
		return err
	}

	return os.WriteFile(backupFile, data, 0644)
}

// LoadPRFormBackup loads the form data from the backup file
func LoadPRFormBackup() (*PRForm, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	backupFile := filepath.Join(homeDir, ".sage", "backups", "pr_form_backup.json")
	data, err := os.ReadFile(backupFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var form PRForm
	if err := json.Unmarshal(data, &form); err != nil {
		return nil, err
	}

	return &form, nil
}

// DeletePRFormBackup deletes the PR form backup file
func DeletePRFormBackup() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	backupFile := filepath.Join(homeDir, ".sage", "backups", "pr_form_backup.json")
	err = os.Remove(backupFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
