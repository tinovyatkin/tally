package rules

import (
	"testing"
)

func TestLintInput_SourceMap(t *testing.T) {
	source := []byte("FROM alpine\nRUN echo hello\nCMD [\"sh\"]")
	input := LintInput{
		Source: source,
	}

	sm := input.SourceMap()
	if sm == nil {
		t.Fatal("SourceMap() returned nil")
	}

	if sm.LineCount() != 3 {
		t.Errorf("LineCount() = %d, want 3", sm.LineCount())
	}

	if sm.Line(0) != "FROM alpine" {
		t.Errorf("Line(0) = %q, want %q", sm.Line(0), "FROM alpine")
	}
}

func TestLintInput_Snippet(t *testing.T) {
	source := []byte("line0\nline1\nline2\nline3\nline4")
	input := LintInput{
		Source: source,
	}

	tests := []struct {
		name      string
		startLine int
		endLine   int
		want      string
	}{
		{"single line", 2, 2, "line2"},
		{"multiple lines", 1, 3, "line1\nline2\nline3"},
		{"all lines", 0, 4, "line0\nline1\nline2\nline3\nline4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := input.Snippet(tt.startLine, tt.endLine)
			if got != tt.want {
				t.Errorf("Snippet(%d, %d) = %q, want %q", tt.startLine, tt.endLine, got, tt.want)
			}
		})
	}
}

func TestLintInput_SnippetForLocation(t *testing.T) {
	source := []byte("line0\nline1\nline2\nline3\nline4")
	input := LintInput{
		Source: source,
	}

	tests := []struct {
		name string
		loc  Location
		want string
	}{
		{
			name: "file level",
			loc:  NewFileLocation("test"),
			want: "",
		},
		{
			name: "point location",
			loc:  NewLineLocation("test", 2),
			want: "line2",
		},
		{
			name: "range same line",
			loc:  NewRangeLocation("test", 1, 0, 1, 5),
			want: "line1",
		},
		{
			name: "range multiple lines",
			loc:  NewRangeLocation("test", 1, 0, 3, 5),
			want: "line1\nline2\nline3",
		},
		{
			name: "range end column 0",
			loc:  NewRangeLocation("test", 1, 0, 3, 0),
			want: "line1\nline2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := input.SnippetForLocation(tt.loc)
			if got != tt.want {
				t.Errorf("SnippetForLocation(%v) = %q, want %q", tt.loc, got, tt.want)
			}
		})
	}
}

func TestLintInput_SnippetForLocation_EmptySource(t *testing.T) {
	input := LintInput{
		Source: []byte{},
	}

	// Should not panic with empty source
	got := input.SnippetForLocation(NewLineLocation("test", 0))
	if got != "" {
		t.Errorf("SnippetForLocation with empty source = %q, want empty", got)
	}
}
