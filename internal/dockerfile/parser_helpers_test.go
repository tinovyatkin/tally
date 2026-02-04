package dockerfile

import (
	"strings"
	"testing"
)

func TestExtractRuleNameFromURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "valid",
			url:  "https://docs.docker.com/go/dockerfile/rule/no-empty-continuation/",
			want: "NoEmptyContinuation",
		},
		{
			name: "valid-no-trailing-slash",
			url:  "https://docs.docker.com/go/dockerfile/rule/no-empty-continuation",
			want: "NoEmptyContinuation",
		},
		{
			name: "wrong-prefix",
			url:  "https://example.com/go/dockerfile/rule/no-empty-continuation/",
			want: "",
		},
		{
			name: "empty-suffix",
			url:  "https://docs.docker.com/go/dockerfile/rule/",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractRuleNameFromURL(tt.url); got != tt.want {
				t.Fatalf("extractRuleNameFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestKebabToPascalCase(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"no-empty-continuation", "NoEmptyContinuation"},
		{"json-args-recommended", "JsonArgsRecommended"},
		{"", ""},
		{"--", ""},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := kebabToPascalCase(tt.in); got != tt.want {
				t.Fatalf("kebabToPascalCase(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestExtractHeredocFiles(t *testing.T) {
	content := syntaxDirective + `FROM alpine
RUN echo hi
COPY <<CONFIG /app/config.txt
key=value
CONFIG
ADD <<DATA /app/data.txt
data
DATA
`

	result, err := Parse(strings.NewReader(content), nil)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	files := ExtractHeredocFiles(result.Stages)
	if !files["CONFIG"] {
		t.Fatalf("expected CONFIG to be detected as heredoc file name")
	}
	if !files["DATA"] {
		t.Fatalf("expected DATA to be detected as heredoc file name")
	}
	if len(files) != 2 {
		t.Fatalf("expected exactly 2 heredoc file names, got %d (%v)", len(files), files)
	}
}
