package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/termchroma"
)

var (
	green  string = ""
	red    string = ""
	yellow string = ""
	blue   string = ""
	white  string = ""
	gray   string = ""
	sage   string = ""

	bold  string = termchroma.Bold
	reset string = termchroma.Reset
)

// Colors
func Green(s string) string  { return green + s + reset }
func Red(s string) string    { return red + s + reset }
func Blue(s string) string   { return blue + s + reset }
func White(s string) string  { return white + s + reset }
func Yellow(s string) string { return yellow + s + reset }
func Gray(s string) string   { return gray + s + reset }
func Sage(s string) string   { return sage + s + reset }
func Bold(s string) string   { return bold + s + reset }

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

// Info prints an info message
func Info(msg string) {
	fmt.Printf("%s %s\n", Blue("•"), msg)
}

// Infof prints a formatted info message
func Infof(format string, args ...interface{}) {
	Info(fmt.Sprintf(format, args...))
}

// Success prints a success message
func Success(msg string) {
	fmt.Printf("%s %s\n", Green("✓"), msg)
}

// Successf prints a formatted success message
func Successf(format string, args ...interface{}) {
	Success(fmt.Sprintf(format, args...))
}

// Warning prints a warning message
func Warning(msg string) {
	fmt.Printf("%s %s\n", Yellow("!"), msg)
}

// Warningf prints a formatted warning message
func Warningf(format string, args ...interface{}) {
	Warning(fmt.Sprintf(format, args...))
}

// Error prints an error message
func Error(msg string) {
	fmt.Printf("%s %s\n", Red("✗"), msg)
}

// Errorf prints a formatted error message
func Errorf(format string, args ...interface{}) {
	Error(fmt.Sprintf(format, args...))
}

// Confirm asks for user confirmation
func Confirm(msg string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s %s [y/N]: ", Yellow("?"), msg)

	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

func nonEmpty(val interface{}) error {
	str, _ := val.(string)
	if str == "" {
		return fmt.Errorf("cannot be empty")
	}
	return nil
}

func ColorHeadings(text string) string {
	headings := []string{
		"Usage:",
		"Examples:",
		"Available Commands:",
		"Flags:",
		"Aliases:",
		"Additional Commands:",
	}

	// Replace each heading with its colorized version.
	for _, heading := range headings {
		text = strings.ReplaceAll(text, heading, fmt.Sprintf("%s%s%s%s", sage, bold, heading, reset))
	}

	text = strings.ReplaceAll(text, "{{rpad .Name .NamePadding }}", fmt.Sprintf("%s%s%s", white, "{{rpad .Name .NamePadding }}", reset))
	text = strings.ReplaceAll(text, "{{.CommandPath}}", fmt.Sprintf("%s%s%s", white, "{{.CommandPath}}", reset))

	return text
}

func init() {
	sage, _ = termchroma.ANSIForeground("#8EA58C")
	blue, _ = termchroma.ANSIForeground("#59B4FF")
	yellow, _ = termchroma.ANSIForeground("#FFC402")
	red, _ = termchroma.ANSIForeground("#FF707E")
	white, _ = termchroma.ANSIForeground("#FFF")
	gray, _ = termchroma.ANSIForeground("#6B737C")
	green, _ = termchroma.ANSIForeground("#98C379")
}
