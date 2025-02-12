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
	if body == "" {
		return ""
	}

	lines := strings.Split(body, "\n")

	// Keep section headers and a few lines after each
	var truncated []string
	inSection := false
	linesAfterHeader := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "##") {
			inSection = true
			linesAfterHeader = 0
			truncated = append(truncated, line)
			continue
		}

		if inSection {
			if linesAfterHeader < 3 && line != "" { // Show up to 3 non-empty lines per section
				if len(line) > maxLineLength {
					truncated = append(truncated, line[:maxLineLength]+"...")
				} else {
					truncated = append(truncated, line)
				}
				linesAfterHeader++
			} else if linesAfterHeader == 3 {
				truncated = append(truncated, "...")
				inSection = false
			}
		}
	}

	if len(truncated) > maxLines {
		truncated = truncated[:maxLines]
		truncated = append(truncated, "...")
	}

	return strings.Join(truncated, "\n")
}

// AskPRForm uses Survey to gather the user's input
// If Body is empty, we'll fetch and use the PR template if available.
func AskPRForm(initial PRForm, ghc gh.Client) (PRForm, error) {
	form := initial

	// Try to get PR template if body is empty
	if form.Body == "" {
		tmpl, err := ghc.GetPRTemplate()
		if err == nil && tmpl != "" {
			// Show preview of the template that will be used
			preview := truncateBody(tmpl, 10, 80)
			fmt.Printf("\nUsing PR Template:\n%s\n\n", preview)
			form.Body = tmpl
		}
	} else {
		// Show preview of existing body (e.g., from AI generation)
		preview := truncateBody(form.Body, 15, 100)
		fmt.Printf("\nProposed PR Description:\n%s\n\n", preview)
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
				Message:       "Pull Request Description (body):",
				FileName:      "*.md",
				Default:       form.Body,
				AppendDefault: true,
				HideDefault:   false,
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
