package ui

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
)

// captureOutput captures stdout during a test
func captureOutput(f func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	outC := make(chan string)
	// Copy stdout to buffer
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	f()

	w.Close()
	os.Stdout = oldStdout

	return <-outC
}

// stripAnsi removes ANSI escape codes from a string
func stripAnsi(str string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return re.ReplaceAllString(str, "")
}

func TestColorFunctions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		colorFn  func(string) string
		contains string
	}{
		{
			name:     "green color",
			input:    "success",
			colorFn:  Green,
			contains: "success",
		},
		{
			name:     "red color",
			input:    "error",
			colorFn:  Red,
			contains: "error",
		},
		{
			name:     "blue color",
			input:    "info",
			colorFn:  Blue,
			contains: "info",
		},
		{
			name:     "white color",
			input:    "text",
			colorFn:  White,
			contains: "text",
		},
		{
			name:     "yellow color",
			input:    "warning",
			colorFn:  Yellow,
			contains: "warning",
		},
		{
			name:     "gray color",
			input:    "disabled",
			colorFn:  Gray,
			contains: "disabled",
		},
		{
			name:     "sage color",
			input:    "brand",
			colorFn:  Sage,
			contains: "brand",
		},
		{
			name:     "bold text",
			input:    "important",
			colorFn:  Bold,
			contains: "important",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.colorFn(tt.input)
			if !strings.Contains(stripAnsi(result), tt.contains) {
				t.Errorf("Expected result to contain %q, got %q", tt.contains, stripAnsi(result))
			}
			if !strings.HasSuffix(result, reset) {
				t.Errorf("Expected result to end with reset sequence")
			}
		})
	}
}

func TestWarnf(t *testing.T) {
	output := captureOutput(func() {
		Warnf("test warning %d", 42)
	})

	expected := "Warning: test warning 42"
	if !strings.Contains(stripAnsi(output), expected) {
		t.Errorf("Expected output to contain %q, got %q", expected, stripAnsi(output))
	}
}

func TestInfoFunctions(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(string)
		input    string
		contains string
	}{
		{
			name:     "info message",
			fn:       Info,
			input:    "test info",
			contains: "ℹ test info",
		},
		{
			name:     "success message",
			fn:       Success,
			input:    "test success",
			contains: "✓ test success",
		},
		{
			name:     "warning message",
			fn:       Warning,
			input:    "test warning",
			contains: "⚠ test warning",
		},
		{
			name:     "error message",
			fn:       Error,
			input:    "test error",
			contains: "✗ test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				tt.fn(tt.input)
			})

			cleanOutput := stripAnsi(output)
			cleanOutput = strings.TrimSpace(cleanOutput)
			if !strings.Contains(cleanOutput, tt.contains) {
				t.Errorf("Expected output to contain %q, got %q", tt.contains, cleanOutput)
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
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "non-empty string",
			input:   "test",
			wantErr: false,
		},
		{
			name:    "non-string value",
			input:   42,
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

func TestColorHeadings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:  "usage heading",
			input: "Usage: command",
			contains: []string{
				"Usage:",
				"command",
			},
		},
		{
			name:  "examples heading",
			input: "Examples:\n  example1\n  example2",
			contains: []string{
				"Examples:",
				"example1",
				"example2",
			},
		},
		{
			name:  "commands heading",
			input: "Available Commands:\n  cmd1\n  cmd2",
			contains: []string{
				"Available Commands:",
				"cmd1",
				"cmd2",
			},
		},
		{
			name:  "flags heading",
			input: "Flags:\n  --flag1\n  --flag2",
			contains: []string{
				"Flags:",
				"--flag1",
				"--flag2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ColorHeadings(tt.input)
			cleanResult := stripAnsi(result)
			for _, s := range tt.contains {
				if !strings.Contains(cleanResult, s) {
					t.Errorf("Expected result to contain %q", s)
				}
			}
		})
	}
}
