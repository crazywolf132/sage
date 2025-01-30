package ui

import (
	"fmt"
	"math"
	"strings"

	"golang.org/x/term"
)

func DrawBox(content string) {
	internalPadding := 6 // Space to the left and right of the text
	internalMargin := 2  // Empty lines above and below the text

	lines := strings.Split(content, "\n")
	maxLength := 0
	for _, line := range lines {
		if len(line) > maxLength {
			maxLength = len(line)
		}
	}
	boxContentWidth := maxLength + 2*internalPadding // Adding internal padding to both sides

	// Calculate left margin for centering the box
	terminalWidth := 80 // Default terminal width
	if width, _, err := term.GetSize(0); err == nil {
		terminalWidth = width
	}
	leftMargin := int(math.Max(float64((terminalWidth-boxContentWidth)/2), 0))
	leftMarginString := strings.Repeat(" ", leftMargin)

	topBorder := ColoredText(leftMarginString+"╭"+strings.Repeat("─", boxContentWidth)+"╮", Sage)
	bottomBorder := ColoredText(leftMarginString+"╰"+strings.Repeat("─", boxContentWidth)+"╯", Sage)

	marginLine := ColoredText(leftMarginString+"│"+strings.Repeat(" ", boxContentWidth)+"│", Sage)
	topMargin := strings.Repeat(marginLine+"\n", internalMargin)
	bottomMargin := strings.Repeat(marginLine+"\n", internalMargin)

	var paddedLines []string
	for _, line := range lines {
		paddingTotal := boxContentWidth - len(line)
		paddingLeft := strings.Repeat(" ", paddingTotal/2)
		paddingRight := strings.Repeat(" ", paddingTotal-len(paddingLeft))
		paddedLine := ColoredText(leftMarginString+"│", Sage) + paddingLeft + line + paddingRight + ColoredText("│", Sage)
		paddedLines = append(paddedLines, paddedLine)
	}

	boxContent := topBorder + "\n" + topMargin + strings.Join(paddedLines, "\n") + "\n" + bottomMargin + bottomBorder

	fmt.Println(boxContent)
}
