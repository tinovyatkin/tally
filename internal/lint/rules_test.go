package lint

import (
	"testing"

	"github.com/tinovyatkin/tally/internal/dockerfile"
)

func TestCheckMaxLines(t *testing.T) {
	tests := []struct {
		name       string
		totalLines int
		maxLines   int
		wantIssue  bool
	}{
		{
			name:       "under limit",
			totalLines: 50,
			maxLines:   100,
			wantIssue:  false,
		},
		{
			name:       "at limit",
			totalLines: 100,
			maxLines:   100,
			wantIssue:  false,
		},
		{
			name:       "over limit",
			totalLines: 150,
			maxLines:   100,
			wantIssue:  true,
		},
		{
			name:       "just over limit",
			totalLines: 101,
			maxLines:   100,
			wantIssue:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &dockerfile.ParseResult{
				TotalLines: tt.totalLines,
			}

			issue := CheckMaxLines(result, tt.maxLines)

			if tt.wantIssue && issue == nil {
				t.Error("expected issue but got nil")
			}
			if !tt.wantIssue && issue != nil {
				t.Errorf("expected no issue but got: %v", issue)
			}
			if issue != nil {
				if issue.Rule != "max-lines" {
					t.Errorf("issue.Rule = %q, want %q", issue.Rule, "max-lines")
				}
				if issue.Severity != "error" {
					t.Errorf("issue.Severity = %q, want %q", issue.Severity, "error")
				}
			}
		})
	}
}
