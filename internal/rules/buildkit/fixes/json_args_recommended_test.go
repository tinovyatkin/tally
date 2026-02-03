package fixes

import (
	"testing"

	"github.com/tinovyatkin/tally/internal/rules"
)

func TestFindDockerfileInlineCommentStart(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantIdx   int
		wantFound bool
	}{
		{
			name:      "no-comment",
			line:      "CMD echo hi",
			wantIdx:   len("CMD echo hi"),
			wantFound: false,
		},
		{
			name:      "inline-comment",
			line:      "CMD echo hi # comment",
			wantIdx:   len("CMD echo hi "),
			wantFound: true,
		},
		{
			name:      "hash-not-comment",
			line:      "CMD echo hi#not-a-comment",
			wantIdx:   len("CMD echo hi#not-a-comment"),
			wantFound: false,
		},
		{
			name:      "hash-in-single-quotes",
			line:      "CMD echo '# not a comment'",
			wantIdx:   len("CMD echo '# not a comment'"),
			wantFound: false,
		},
		{
			name:      "hash-in-double-quotes",
			line:      `CMD echo "# not a comment"`,
			wantIdx:   len(`CMD echo "# not a comment"`),
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIdx, gotFound := findDockerfileInlineCommentStart([]byte(tt.line))
			if gotFound != tt.wantFound {
				t.Fatalf("found = %v, want %v (idx=%d)", gotFound, tt.wantFound, gotIdx)
			}
			if gotIdx != tt.wantIdx {
				t.Fatalf("idx = %d, want %d", gotIdx, tt.wantIdx)
			}
		})
	}
}

func TestEnrichJSONArgsRecommendedFix_SkipsComplexShell(t *testing.T) {
	df := "FROM alpine\nCMD echo *.txt\n"
	source := []byte(df)

	v := rules.NewViolation(
		rules.NewLineLocation("Dockerfile", 2),
		"buildkit/JSONArgsRecommended",
		"msg",
		rules.SeverityInfo,
	)
	// Make the location look like an instruction location (line range), not a point.
	v.Location = rules.NewRangeLocation("Dockerfile", 2, 0, 2, len("CMD echo *.txt"))

	enrichJSONArgsRecommendedFix(&v, source)
	if v.SuggestedFix != nil {
		t.Fatalf("expected no fix for globbing shell command, got %+v", v.SuggestedFix)
	}
}
