package ui

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
)

// Colors
func Green(s string) string  { return "\033[32m" + s + "\033[0m" }
func Red(s string) string    { return "\033[31m" + s + "\033[0m" }
func Yellow(s string) string { return "\033[33m" + s + "\033[0m" }
func Gray(s string) string   { return "\033[90m" + s + "\033[0m" }
func Sage(s string) string   { return "\033[38;5;114m" + s + "\033[0m" }
func Bold(s string) string   { return "\033[1m" + s + "\033[0m" }

// Logging
func Warnf(format string, args ...interface{}) {
	fmt.Fprintf(stderr, Red("Warning: ")+format, args...)
}

var stderr = getStderr()

func getStderr() *ColoredWriter {
	return &ColoredWriter{}
}

type ColoredWriter struct{}

func (c *ColoredWriter) Write(p []byte) (n int, err error) {
	return fmt.Print(Red(string(p)))
}

// commit message prompt
func AskCommitMessage(useConventional bool) (msg string, scope string, ctype string, err error) {
	if !useConventional {
		err = survey.AskOne(&survey.Input{Message: "Commit message:"}, &msg, survey.WithValidator(nonEmpty))
		return msg, "", "", err
	}

	types := []string{"feat", "fix", "docs", "style", "refactor", "test", "chore"}
	err = survey.Ask([]*survey.Question{
		{
			Name: "type",
			Prompt: &survey.Select{
				Message: "Commit type:",
				Options: types,
			},
			Validate: nonEmpty,
		},
		{
			Name:   "scope",
			Prompt: &survey.Input{Message: "Scope (optional):"},
		},
		{
			Name: "msg",
			Prompt: &survey.Input{
				Message: "Commit message:",
			},
			Validate: nonEmpty,
		},
	}, &struct {
		Type  string
		Scope string
		Msg   string
	}{}, survey.WithValidator(nonEmpty))

	// The above is a bit hacky with the ephemeral struct. Let's do a simpler approach:
	var form struct {
		Type  string
		Scope string
		Msg   string
	}
	if err == nil {
		err = survey.Ask([]*survey.Question{
			{
				Name:     "type",
				Prompt:   &survey.Select{Message: "Type:", Options: types},
				Validate: nonEmpty,
			},
			{Name: "scope", Prompt: &survey.Input{Message: "Scope:"}},
			{Name: "msg", Prompt: &survey.Input{Message: "Message:"}, Validate: nonEmpty},
		}, &form)
	}
	return form.Msg, form.Scope, form.Type, err
}

func nonEmpty(val interface{}) error {
	str, _ := val.(string)
	if str == "" {
		return fmt.Errorf("cannot be empty")
	}
	return nil
}
