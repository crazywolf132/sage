package ui

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/sage/internal/gh"
)

// PRForm is the structure we use to gather from the user.
type PRForm struct {
	Title     string
	Body      string
	Base      string
	Draft     bool
	Labels    []string
	Reviewers []string
}

// truncateBody returns a truncated version of the body text suitable for preview
func truncateBody(body string, maxLines int, maxLineLength int) string {
	lines := strings.Split(body, "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines = append(lines, "...")
	}

	// Truncate each line if too long
	for i, line := range lines {
		if len(line) > maxLineLength {
			lines[i] = line[:maxLineLength] + "..."
		}
	}

	return strings.Join(lines, "\n")
}

// AskPRForm uses Survey to gather the user's input
// If Body is empty, we'll fetch and use the PR template if available.
func AskPRForm(initial PRForm, ghc gh.Client) (PRForm, error) {
	form := initial

	// Try to get PR template if body is empty
	if form.Body == "" {
		tmpl, _ := ghc.GetPRTemplate()
		if tmpl != "" {
			// Show preview of the template that will be used
			preview := truncateBody(tmpl, 10, 80)
			fmt.Printf("\nUsing PR Template:\n%s\n\n", preview)
			form.Body = tmpl
		}
	}

	qs := []*survey.Question{
		{
			Name: "Title",
			Prompt: &survey.Input{
				Message: "Pull Request Title:",
				Default: form.Title,
			},
			Validate: nonEmpty,
		},
		{
			Name: "Body",
			Prompt: &survey.Editor{
				Message:  "Pull Request Description (body):",
				FileName: "PR_BODY*.md",
				Default:  form.Body,
			},
		},
		{
			Name: "Base",
			Prompt: &survey.Input{
				Message: "Base branch:",
				Default: nonEmptyOr(form.Base, "main"),
			},
		},
		{
			Name: "Draft",
			Prompt: &survey.Confirm{
				Message: "Create as draft?",
				Default: form.Draft,
			},
		},
		{
			Name: "Labels",
			Prompt: &survey.Input{
				Message: "Labels (comma separated):",
				Default: strings.Join(form.Labels, ","),
			},
		},
		{
			Name: "Reviewers",
			Prompt: &survey.Input{
				Message: "Reviewers (comma separated usernames):",
				Default: strings.Join(form.Reviewers, ","),
			},
		},
	}

	var answers struct {
		Title     string
		Body      string
		Base      string
		Draft     bool
		Labels    string
		Reviewers string
	}
	err := survey.Ask(qs, &answers)
	if err != nil {
		return form, err
	}

	// Convert
	form.Title = answers.Title
	form.Body = answers.Body
	form.Base = answers.Base
	form.Draft = answers.Draft
	if answers.Labels != "" {
		form.Labels = splitTrim(answers.Labels, ",")
	}
	if answers.Reviewers != "" {
		form.Reviewers = splitTrim(answers.Reviewers, ",")
	}
	return form, nil
}

func nonEmptyOr(val, fallback string) string {
	v := strings.TrimSpace(val)
	if v == "" {
		return fallback
	}
	return v
}

func splitTrim(s, sep string) []string {
	var out []string
	for _, part := range strings.Split(s, sep) {
		trim := strings.TrimSpace(part)
		if trim != "" {
			out = append(out, trim)
		}
	}
	return out
}
