package ui

import (
	"fmt"
	"strings"

	"github.com/crazywolf132/termchroma"
)

var (
	Sage   string = ""
	Blue   string = ""
	Red    string = ""
	Yellow string = ""
	White  string = ""

	Reset string = termchroma.Reset
	Bold  string = termchroma.Bold
)

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
		text = strings.ReplaceAll(text, heading, fmt.Sprintf("%s%s%s%s", Sage, Bold, heading, Reset))
	}

	return text
}

// ColoredText returns text in the specified color
func ColoredText(text string, color string) string {
	return fmt.Sprintf("%s%s%s", color, text, Reset)
}

// SuccessText returns text in green color
func SuccessText(text string) string {
	return ColoredText(text, Sage)
}

// WarningText returns text in yellow color
func WarningText(text string) string {
	return ColoredText(text, Yellow)
}

// ErrorText returns text in red color
func ErrorText(text string) string {
	return ColoredText(text, Red)
}

// InfoText returns text in blue color
func InfoText(text string) string {
	return ColoredText(text, Blue)
}

func init() {
	Sage, _ = termchroma.ANSIForeground("#8EA58C")
	Blue, _ = termchroma.ANSIForeground("#59B4FF")
	Yellow, _ = termchroma.ANSIForeground("#FFC402")
	Red, _ = termchroma.ANSIForeground("#FF707E")
	White, _ = termchroma.ANSIForeground("#FFF")
}
