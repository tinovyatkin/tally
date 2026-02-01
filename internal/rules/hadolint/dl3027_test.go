package hadolint

import (
	"testing"

	"github.com/tinovyatkin/tally/internal/testutil"
)

func TestDL3027Rule_Check(t *testing.T) {
	tests := []struct {
		name       string
		dockerfile string
		wantCount  int
	}{
		// Original Hadolint test cases - should trigger
		{
			name: "apt install command",
			dockerfile: `FROM ubuntu
RUN apt install python`,
			wantCount: 1,
		},

		// Additional test cases for comprehensive coverage
		{
			name: "apt-get install should not trigger",
			dockerfile: `FROM ubuntu
RUN apt-get install python`,
			wantCount: 0,
		},
		{
			name: "apt-cache should not trigger",
			dockerfile: `FROM ubuntu
RUN apt-cache search python`,
			wantCount: 0,
		},
		{
			name: "apt update command",
			dockerfile: `FROM ubuntu
RUN apt update`,
			wantCount: 1,
		},
		{
			name: "apt upgrade command",
			dockerfile: `FROM ubuntu
RUN apt upgrade`,
			wantCount: 1,
		},
		{
			name: "apt with full path",
			dockerfile: `FROM ubuntu
RUN /usr/bin/apt install python`,
			wantCount: 1,
		},
		{
			name: "apt in command chain",
			dockerfile: `FROM ubuntu
RUN apt update && apt install python`,
			wantCount: 1, // Both should be detected, but we count violations per RUN
		},
		{
			name: "apt with sudo",
			dockerfile: `FROM ubuntu
RUN sudo apt install python`,
			// Note: sudo is intentionally not a transparent wrapper in our shell parser
			// so that DL3004 can detect it. In this case, only sudo is detected, not apt.
			// This is a design trade-off: detecting both commands would require sudo
			// to be in commandWrappers, but then DL3004 checking would need adjustment.
			wantCount: 0,
		},
		{
			name: "apt with env wrapper",
			dockerfile: `FROM ubuntu
RUN env DEBIAN_FRONTEND=noninteractive apt install python`,
			wantCount: 1,
		},
		{
			name: "apt in shell -c",
			dockerfile: `FROM ubuntu
RUN sh -c 'apt install python'`,
			wantCount: 1,
		},
		{
			name: "word 'apt' in string should not trigger",
			dockerfile: `FROM ubuntu
RUN echo "adapt to changes"`,
			wantCount: 0,
		},
		{
			name: "word 'apt' in package name should not trigger",
			dockerfile: `FROM ubuntu
RUN apt-get install aptitude`,
			wantCount: 0,
		},
		{
			name: "multiple RUN commands with apt",
			dockerfile: `FROM ubuntu
RUN apt update
RUN apt install python
RUN apt upgrade`,
			wantCount: 3,
		},
		{
			name: "apt in exec form",
			dockerfile: `FROM ubuntu
RUN ["apt", "install", "python"]`,
			wantCount: 1,
		},
		{
			name: "multi-stage with apt in one stage",
			dockerfile: `FROM ubuntu AS builder
RUN apt install python

FROM alpine
RUN apk add python`,
			wantCount: 1,
		},
		{
			name: "ONBUILD with apt",
			dockerfile: `FROM ubuntu
ONBUILD RUN apt install python`,
			wantCount: 0, // ONBUILD not yet supported in our implementation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := testutil.MakeLintInput(t, "Dockerfile", tt.dockerfile)
			r := NewDL3027Rule()
			violations := r.Check(input)

			if len(violations) != tt.wantCount {
				t.Errorf("got %d violations, want %d", len(violations), tt.wantCount)
				for i, v := range violations {
					t.Logf("violation %d: %s at %v", i+1, v.Message, v.Location)
				}
			}

			// Verify violation details for positive cases
			if tt.wantCount > 0 && len(violations) > 0 {
				v := violations[0]
				if v.RuleCode != "hadolint/DL3027" {
					t.Errorf("got rule code %q, want %q", v.RuleCode, "hadolint/DL3027")
				}
				if v.Message == "" {
					t.Error("violation message is empty")
				}
				if v.Detail == "" {
					t.Error("violation detail is empty")
				}
				if v.DocURL != "https://github.com/hadolint/hadolint/wiki/DL3027" {
					t.Errorf("got doc URL %q, want %q", v.DocURL, "https://github.com/hadolint/hadolint/wiki/DL3027")
				}
			}
		})
	}
}

func TestDL3027Rule_Metadata(t *testing.T) {
	r := NewDL3027Rule()
	meta := r.Metadata()

	if meta.Code != "hadolint/DL3027" {
		t.Errorf("got code %q, want %q", meta.Code, "hadolint/DL3027")
	}
	if meta.Name == "" {
		t.Error("name is empty")
	}
	if meta.Description == "" {
		t.Error("description is empty")
	}
	if meta.DocURL != "https://github.com/hadolint/hadolint/wiki/DL3027" {
		t.Errorf("got doc URL %q, want %q", meta.DocURL, "https://github.com/hadolint/hadolint/wiki/DL3027")
	}
	if meta.Category != "style" {
		t.Errorf("got category %q, want %q", meta.Category, "style")
	}
}
