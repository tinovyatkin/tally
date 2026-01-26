package dockerfile

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name             string
		content          string
		expectedLines    int
		expectedBlank    int
		expectedComments int
	}{
		{
			name:             "simple dockerfile",
			content:          "FROM alpine:3.18\nRUN echo hello\n",
			expectedLines:    2,
			expectedBlank:    0,
			expectedComments: 0,
		},
		{
			name:             "multiline dockerfile",
			content:          "FROM alpine:3.18\nRUN apk add --no-cache \\\n    curl \\\n    wget\nCMD [\"sh\"]\n",
			expectedLines:    5,
			expectedBlank:    0,
			expectedComments: 0,
		},
		{
			name:             "single line no newline",
			content:          "FROM alpine:3.18",
			expectedLines:    1,
			expectedBlank:    0,
			expectedComments: 0,
		},
		{
			name:             "empty lines",
			content:          "FROM alpine:3.18\n\n\nRUN echo hello\n",
			expectedLines:    4,
			expectedBlank:    2,
			expectedComments: 0,
		},
		{
			name:             "with comments",
			content:          "# This is a comment\nFROM alpine:3.18\n# Another comment\nRUN echo hello\n",
			expectedLines:    4,
			expectedBlank:    0,
			expectedComments: 2,
		},
		{
			name:             "mixed blanks and comments",
			content:          "# Header comment\n\nFROM alpine:3.18\n\n# Install packages\nRUN apk add curl\n",
			expectedLines:    6,
			expectedBlank:    2,
			expectedComments: 2,
		},
		{
			name:             "whitespace-only lines count as blank",
			content:          "FROM alpine:3.18\n   \n\t\nRUN echo hello\n",
			expectedLines:    4,
			expectedBlank:    2,
			expectedComments: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
			if err := os.WriteFile(dockerfilePath, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}

			result, err := ParseFile(context.Background(), dockerfilePath)
			if err != nil {
				t.Fatalf("ParseFile() error = %v", err)
			}

			if result.TotalLines != tt.expectedLines {
				t.Errorf("TotalLines = %d, want %d", result.TotalLines, tt.expectedLines)
			}

			if result.BlankLines != tt.expectedBlank {
				t.Errorf("BlankLines = %d, want %d", result.BlankLines, tt.expectedBlank)
			}

			if result.CommentLines != tt.expectedComments {
				t.Errorf("CommentLines = %d, want %d", result.CommentLines, tt.expectedComments)
			}
		})
	}
}

func TestParse_Stages(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		expectedStages int
		stageNames     []string
	}{
		{
			name:           "single stage",
			content:        "FROM alpine:3.18\nRUN echo hello\n",
			expectedStages: 1,
			stageNames:     []string{""},
		},
		{
			name:           "named single stage",
			content:        "FROM alpine:3.18 AS builder\nRUN echo hello\n",
			expectedStages: 1,
			stageNames:     []string{"builder"},
		},
		{
			name:           "multi-stage build",
			content:        "FROM golang:1.21 AS builder\nRUN go build\n\nFROM alpine:3.18\nCOPY --from=builder /app /app\n",
			expectedStages: 2,
			stageNames:     []string{"builder", ""},
		},
		{
			name: "three named stages",
			content: "FROM node:20 AS deps\nRUN npm ci\n\n" +
				"FROM node:20 AS builder\nCOPY --from=deps /app/node_modules ./node_modules\n" +
				"RUN npm run build\n\nFROM node:20-slim AS runtime\nCOPY --from=builder /app/dist ./dist\n",
			expectedStages: 3,
			stageNames:     []string{"deps", "builder", "runtime"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
			if err := os.WriteFile(dockerfilePath, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}

			result, err := ParseFile(context.Background(), dockerfilePath)
			if err != nil {
				t.Fatalf("ParseFile() error = %v", err)
			}

			if len(result.Stages) != tt.expectedStages {
				t.Errorf("len(Stages) = %d, want %d", len(result.Stages), tt.expectedStages)
			}

			for i, name := range tt.stageNames {
				if i < len(result.Stages) && result.Stages[i].Name != name {
					t.Errorf("Stages[%d].Name = %q, want %q", i, result.Stages[i].Name, name)
				}
			}
		})
	}
}

func TestParse_MetaArgs(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		expectedArgs []string
	}{
		{
			name:         "no meta args",
			content:      "FROM alpine:3.18\nRUN echo hello\n",
			expectedArgs: nil,
		},
		{
			name:         "single meta arg",
			content:      "ARG VERSION=1.0\nFROM alpine:${VERSION}\n",
			expectedArgs: []string{"VERSION"},
		},
		{
			name:         "multiple meta args",
			content:      "ARG BASE_IMAGE=alpine\nARG VERSION=3.18\nFROM ${BASE_IMAGE}:${VERSION}\n",
			expectedArgs: []string{"BASE_IMAGE", "VERSION"},
		},
		{
			name:         "args after FROM are not meta args",
			content:      "ARG VERSION=1.0\nFROM alpine:${VERSION}\nARG BUILD_TYPE=release\nRUN echo $BUILD_TYPE\n",
			expectedArgs: []string{"VERSION"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
			if err := os.WriteFile(dockerfilePath, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}

			result, err := ParseFile(context.Background(), dockerfilePath)
			if err != nil {
				t.Fatalf("ParseFile() error = %v", err)
			}

			if len(result.MetaArgs) != len(tt.expectedArgs) {
				t.Errorf("len(MetaArgs) = %d, want %d", len(result.MetaArgs), len(tt.expectedArgs))
			}

			for i, name := range tt.expectedArgs {
				if i < len(result.MetaArgs) && len(result.MetaArgs[i].Args) > 0 {
					if result.MetaArgs[i].Args[0].Key != name {
						t.Errorf("MetaArgs[%d].Args[0].Key = %q, want %q", i, result.MetaArgs[i].Args[0].Key, name)
					}
				}
			}
		})
	}
}

func TestParse_BuildKitWarnings(t *testing.T) {
	tests := []struct {
		name             string
		content          string
		expectedWarnings int
		wantRuleName     string
	}{
		{
			name:             "no warnings",
			content:          "FROM alpine:3.18\nRUN echo hello\n",
			expectedWarnings: 0,
		},
		{
			name:             "MAINTAINER deprecated",
			content:          "FROM alpine:3.18\nMAINTAINER test@example.com\n",
			expectedWarnings: 1,
			wantRuleName:     "MaintainerDeprecated",
		},
		{
			name:             "stage name casing",
			content:          "FROM alpine:3.18 AS Builder\nRUN echo hello\n",
			expectedWarnings: 1,
			wantRuleName:     "StageNameCasing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
			if err := os.WriteFile(dockerfilePath, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}

			result, err := ParseFile(context.Background(), dockerfilePath)
			if err != nil {
				t.Fatalf("ParseFile() error = %v", err)
			}

			if len(result.Warnings) != tt.expectedWarnings {
				t.Errorf("len(Warnings) = %d, want %d", len(result.Warnings), tt.expectedWarnings)
				for i, w := range result.Warnings {
					t.Logf("  Warning[%d]: %s - %s", i, w.RuleName, w.Message)
				}
			}

			if tt.wantRuleName != "" && len(result.Warnings) > 0 {
				if result.Warnings[0].RuleName != tt.wantRuleName {
					t.Errorf("Warnings[0].RuleName = %q, want %q", result.Warnings[0].RuleName, tt.wantRuleName)
				}
			}
		})
	}
}

func TestParse_Source(t *testing.T) {
	content := "FROM alpine:3.18\nRUN echo hello\n"
	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ParseFile(context.Background(), dockerfilePath)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	if string(result.Source) != content {
		t.Errorf("Source = %q, want %q", string(result.Source), content)
	}
}

func TestCountLines(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectedLines int
	}{
		{
			name:          "two lines with newline",
			content:       "line1\nline2\n",
			expectedLines: 2,
		},
		{
			name:          "two lines no trailing newline",
			content:       "line1\nline2",
			expectedLines: 2,
		},
		{
			name:          "single line",
			content:       "line1\n",
			expectedLines: 1,
		},
		{
			name:          "empty file",
			content:       "",
			expectedLines: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "test.txt")
			if err := os.WriteFile(filePath, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}

			lines, err := CountLines(filePath)
			if err != nil {
				t.Fatalf("CountLines() error = %v", err)
			}

			if lines != tt.expectedLines {
				t.Errorf("CountLines() = %d, want %d", lines, tt.expectedLines)
			}
		})
	}
}
