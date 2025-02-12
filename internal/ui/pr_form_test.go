package ui

import (
	"testing"
)

func TestTruncateBody(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		maxLines      int
		maxLineLength int
		want          string
	}{
		{
			name:          "Short text unchanged",
			body:          "Short text",
			maxLines:      5,
			maxLineLength: 20,
			want:          "Short text",
		},
		{
			name:          "Long line truncated",
			body:          "This is a very long line that should be truncated",
			maxLines:      1,
			maxLineLength: 20,
			want:          "This is a very long ...",
		},
		{
			name:          "Multiple lines truncated",
			body:          "Line 1\nLine 2\nLine 3\nLine 4\nLine 5",
			maxLines:      3,
			maxLineLength: 20,
			want:          "Line 1\nLine 2\nLine 3\n...",
		},
		{
			name:          "Both lines and length truncated",
			body:          "Very long line 1\nVery long line 2\nVery long line 3\nVery long line 4",
			maxLines:      2,
			maxLineLength: 10,
			want:          "Very long ...\nVery long ...\n...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateBody(tt.body, tt.maxLines, tt.maxLineLength)
			if got != tt.want {
				t.Errorf("truncateBody() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSplitTrim(t *testing.T) {
	tests := []struct {
		name string
		s    string
		sep  string
		want []string
	}{
		{
			name: "Simple split",
			s:    "a,b,c",
			sep:  ",",
			want: []string{"a", "b", "c"},
		},
		{
			name: "With spaces",
			s:    "a, b,  c",
			sep:  ",",
			want: []string{"a", "b", "c"},
		},
		{
			name: "Empty parts removed",
			s:    "a,,b, ,c",
			sep:  ",",
			want: []string{"a", "b", "c"},
		},
		{
			name: "Different separator",
			s:    "a;b; c",
			sep:  ";",
			want: []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitTrim(tt.s, tt.sep)
			if len(got) != len(tt.want) {
				t.Errorf("splitTrim() returned %d items, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitTrim()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestNonEmptyOr(t *testing.T) {
	tests := []struct {
		name     string
		val      string
		fallback string
		want     string
	}{
		{
			name:     "Empty value uses fallback",
			val:      "",
			fallback: "default",
			want:     "default",
		},
		{
			name:     "Non-empty value used",
			val:      "actual",
			fallback: "default",
			want:     "actual",
		},
		{
			name:     "Whitespace value uses fallback",
			val:      "   ",
			fallback: "default",
			want:     "default",
		},
		{
			name:     "Empty fallback with empty value",
			val:      "",
			fallback: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nonEmptyOr(tt.val, tt.fallback)
			if got != tt.want {
				t.Errorf("nonEmptyOr() = %q, want %q", got, tt.want)
			}
		})
	}
}
