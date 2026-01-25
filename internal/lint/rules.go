package lint

import (
	"fmt"

	"github.com/tinovyatkin/tally/internal/dockerfile"
)

// Issue represents a linting issue found in a Dockerfile
type Issue struct {
	// Rule is the rule identifier (e.g., "max-lines")
	Rule string `json:"rule"`
	// Line is the line number where the issue was found (0 for file-level issues)
	Line int `json:"line"`
	// Message is the human-readable description of the issue
	Message string `json:"message"`
	// Severity is the issue severity (error, warning, info)
	Severity string `json:"severity"`
}

// FileResult contains the linting results for a single file
type FileResult struct {
	// File is the path to the Dockerfile
	File string `json:"file"`
	// Lines is the total number of lines in the file
	Lines int `json:"lines"`
	// Issues is the list of linting issues found
	Issues []Issue `json:"issues"`
}

// CheckMaxLines checks if the Dockerfile exceeds the maximum line count
func CheckMaxLines(result *dockerfile.ParseResult, maxLines int) *Issue {
	if result.TotalLines > maxLines {
		return &Issue{
			Rule:     "max-lines",
			Line:     0, // File-level issue
			Message:  fmt.Sprintf("file has %d lines, maximum allowed is %d", result.TotalLines, maxLines),
			Severity: "error",
		}
	}
	return nil
}
