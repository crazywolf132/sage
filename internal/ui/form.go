package ui

import (
	"fmt"

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

// GetPRDetails prompts the user for pull request details
func GetPRDetails(template string) (PRForm, error) {
	var form PRForm
	form.Body = template // Set initial value before creating the form

	err := huh.NewForm(
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
			huh.NewText().
				Title("Body").
				Value(&form.Body).
				CharLimit(4000).
				Validate(func(s string) error {
					if len(s) < 10 {
						return fmt.Errorf("PR description should be at least 10 characters")
					}
					return nil
				}),
		),
	).Run()

	return form, err
}
