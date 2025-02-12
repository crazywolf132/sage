package ui

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/crazywolf132/termchroma"
)

func TestColorFormatting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		colorFn  func(string) string
		contains []string // Strings that should be present in the output
	}{
		{
			name:    "Green formatting",
			input:   "success",
			colorFn: Green,
			contains: []string{
				"success",        // Original text
				"\x1b[",          // ANSI escape sequence start
				"m",              // ANSI escape sequence end
				termchroma.Reset, // Reset sequence
			},
		},
		{
			name:    "Red formatting",
			input:   "error",
			colorFn: Red,
			contains: []string{
				"error",
				"\x1b[",
				"m",
				termchroma.Reset,
			},
		},
		{
			name:    "Blue formatting",
			input:   "info",
			colorFn: Blue,
			contains: []string{
				"info",
				"\x1b[",
				"m",
				termchroma.Reset,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.colorFn(tt.input)
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("Expected output to contain %q, but it didn't: %q", substr, result)
				}
			}
		})
	}
}

func TestMessageFunctions(t *testing.T) {
	// Redirect stdout to capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	tests := []struct {
		name     string
		fn       func(string)
		input    string
		contains []string
	}{
		{
			name:  "Info message",
			fn:    Info,
			input: "test info",
			contains: []string{
				"ℹ",
				"test info",
			},
		},
		{
			name:  "Success message",
			fn:    Success,
			input: "test success",
			contains: []string{
				"✓",
				"test success",
			},
		},
		{
			name:  "Warning message",
			fn:    Warning,
			input: "test warning",
			contains: []string{
				"⚠",
				"test warning",
			},
		},
		{
			name:  "Error message",
			fn:    Error,
			input: "test error",
			contains: []string{
				"✗",
				"test error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn(tt.input)
			w.Close()
			out, _ := io.ReadAll(r)
			output := string(out)
			for _, substr := range tt.contains {
				if !strings.Contains(output, substr) {
					t.Errorf("Expected output to contain %q, but it didn't: %q", substr, output)
				}
			}
			// Reset for next test
			r, w, _ = os.Pipe()
			os.Stdout = w
		})
	}

	// Restore stdout
	os.Stdout = oldStdout
}

func TestColorHeadings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:  "Usage heading",
			input: "Usage:",
			contains: []string{
				sage,
				bold,
				"Usage:",
				reset,
			},
		},
		{
			name:  "Examples heading",
			input: "Examples:",
			contains: []string{
				sage,
				bold,
				"Examples:",
				reset,
			},
		},
		{
			name:  "Multiple headings",
			input: "Usage:\nExamples:\nFlags:",
			contains: []string{
				"Usage:",
				"Examples:",
				"Flags:",
				sage,
				bold,
				reset,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ColorHeadings(tt.input)
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("Expected output to contain %q, but it didn't: %q", substr, result)
				}
			}
		})
	}
}

func TestNonEmpty(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name:    "Empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "Non-empty string",
			input:   "test",
			wantErr: false,
		},
		{
			name:    "Non-string input",
			input:   123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := nonEmpty(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("nonEmpty() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
