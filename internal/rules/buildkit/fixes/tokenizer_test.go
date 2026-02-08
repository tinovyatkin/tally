package fixes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenizeLine(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		line     string
		expected []Token
	}{
		{
			name: "simple FROM",
			line: "FROM alpine:3.18",
			expected: []Token{
				{Type: TokenKeyword, Value: "FROM", Start: 0, End: 4},
				{Type: TokenWhitespace, Value: " ", Start: 4, End: 5},
				{Type: TokenArgument, Value: "alpine:3.18", Start: 5, End: 16},
			},
		},
		{
			name: "FROM with AS",
			line: "FROM alpine:3.18 AS builder",
			expected: []Token{
				{Type: TokenKeyword, Value: "FROM", Start: 0, End: 4},
				{Type: TokenWhitespace, Value: " ", Start: 4, End: 5},
				{Type: TokenArgument, Value: "alpine:3.18", Start: 5, End: 16},
				{Type: TokenWhitespace, Value: " ", Start: 16, End: 17},
				{Type: TokenKeyword, Value: "AS", Start: 17, End: 19},
				{Type: TokenWhitespace, Value: " ", Start: 19, End: 20},
				{Type: TokenArgument, Value: "builder", Start: 20, End: 27},
			},
		},
		{
			name: "FROM with platform flag",
			line: "FROM --platform=linux/amd64 alpine AS builder",
			expected: []Token{
				{Type: TokenKeyword, Value: "FROM", Start: 0, End: 4},
				{Type: TokenWhitespace, Value: " ", Start: 4, End: 5},
				{Type: TokenFlag, Value: "--platform=linux/amd64", Start: 5, End: 27},
				{Type: TokenWhitespace, Value: " ", Start: 27, End: 28},
				{Type: TokenArgument, Value: "alpine", Start: 28, End: 34},
				{Type: TokenWhitespace, Value: " ", Start: 34, End: 35},
				{Type: TokenKeyword, Value: "AS", Start: 35, End: 37},
				{Type: TokenWhitespace, Value: " ", Start: 37, End: 38},
				{Type: TokenArgument, Value: "builder", Start: 38, End: 45},
			},
		},
		{
			name: "COPY with --from",
			line: "COPY --from=builder /app /app",
			expected: []Token{
				{Type: TokenKeyword, Value: "COPY", Start: 0, End: 4},
				{Type: TokenWhitespace, Value: " ", Start: 4, End: 5},
				{Type: TokenFlag, Value: "--from=builder", Start: 5, End: 19},
				{Type: TokenWhitespace, Value: " ", Start: 19, End: 20},
				{Type: TokenArgument, Value: "/app", Start: 20, End: 24},
				{Type: TokenWhitespace, Value: " ", Start: 24, End: 25},
				{Type: TokenArgument, Value: "/app", Start: 25, End: 29},
			},
		},
		{
			name: "COPY with quoted --from",
			line: `COPY --from="my stage" /app /app`,
			expected: []Token{
				{Type: TokenKeyword, Value: "COPY", Start: 0, End: 4},
				{Type: TokenWhitespace, Value: " ", Start: 4, End: 5},
				{Type: TokenFlag, Value: `--from="my stage"`, Start: 5, End: 22},
				{Type: TokenWhitespace, Value: " ", Start: 22, End: 23},
				{Type: TokenArgument, Value: "/app", Start: 23, End: 27},
				{Type: TokenWhitespace, Value: " ", Start: 27, End: 28},
				{Type: TokenArgument, Value: "/app", Start: 28, End: 32},
			},
		},
		{
			name: "multiple whitespace",
			line: "FROM   alpine",
			expected: []Token{
				{Type: TokenKeyword, Value: "FROM", Start: 0, End: 4},
				{Type: TokenWhitespace, Value: "   ", Start: 4, End: 7},
				{Type: TokenArgument, Value: "alpine", Start: 7, End: 13},
			},
		},
		{
			name: "tabs and spaces",
			line: "FROM\t alpine",
			expected: []Token{
				{Type: TokenKeyword, Value: "FROM", Start: 0, End: 4},
				{Type: TokenWhitespace, Value: "\t ", Start: 4, End: 6},
				{Type: TokenArgument, Value: "alpine", Start: 6, End: 12},
			},
		},
		{
			name: "lowercase as keyword",
			line: "from alpine as builder",
			expected: []Token{
				{Type: TokenKeyword, Value: "from", Start: 0, End: 4},
				{Type: TokenWhitespace, Value: " ", Start: 4, End: 5},
				{Type: TokenArgument, Value: "alpine", Start: 5, End: 11},
				{Type: TokenWhitespace, Value: " ", Start: 11, End: 12},
				{Type: TokenKeyword, Value: "as", Start: 12, End: 14},
				{Type: TokenWhitespace, Value: " ", Start: 14, End: 15},
				{Type: TokenArgument, Value: "builder", Start: 15, End: 22},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tokens := TokenizeLine([]byte(tt.line))
			assert.Equal(t, tt.expected, tokens)
		})
	}
}

func TestInstructionTokens_FindKeyword(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		line    string
		keyword string
		want    *Token
	}{
		{
			name:    "find AS in FROM",
			line:    "FROM alpine AS builder",
			keyword: "AS",
			want:    &Token{Type: TokenKeyword, Value: "AS", Start: 12, End: 14},
		},
		{
			name:    "find as lowercase",
			line:    "from alpine as builder",
			keyword: "AS",
			want:    &Token{Type: TokenKeyword, Value: "as", Start: 12, End: 14},
		},
		{
			name:    "AS not found",
			line:    "FROM alpine",
			keyword: "AS",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			it := ParseInstruction([]byte(tt.line))
			got := it.FindKeyword(tt.keyword)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInstructionTokens_FindFlag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		line string
		flag string
		want *Token
	}{
		{
			name: "find --from",
			line: "COPY --from=builder /app /app",
			flag: "from",
			want: &Token{Type: TokenFlag, Value: "--from=builder", Start: 5, End: 19},
		},
		{
			name: "find --from uppercase search",
			line: "COPY --from=builder /app /app",
			flag: "FROM",
			want: &Token{Type: TokenFlag, Value: "--from=builder", Start: 5, End: 19},
		},
		{
			name: "find --platform",
			line: "FROM --platform=linux/amd64 alpine",
			flag: "platform",
			want: &Token{Type: TokenFlag, Value: "--platform=linux/amd64", Start: 5, End: 27},
		},
		{
			name: "flag not found",
			line: "COPY /app /app",
			flag: "from",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			it := ParseInstruction([]byte(tt.line))
			got := it.FindFlag(tt.flag)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInstructionTokens_FlagValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		line      string
		flag      string
		wantStart int
		wantEnd   int
		wantValue string
	}{
		{
			name:      "simple flag value",
			line:      "COPY --from=builder /app /app",
			flag:      "from",
			wantStart: 12,
			wantEnd:   19,
			wantValue: "builder",
		},
		{
			name:      "quoted flag value",
			line:      `COPY --from="my stage" /app /app`,
			flag:      "from",
			wantStart: 13,
			wantEnd:   21,
			wantValue: "my stage",
		},
		{
			name:      "single quoted flag value",
			line:      `COPY --from='my stage' /app /app`,
			flag:      "from",
			wantStart: 13,
			wantEnd:   21,
			wantValue: "my stage",
		},
		{
			name:      "flag not found",
			line:      "COPY /app /app",
			flag:      "from",
			wantStart: -1,
			wantEnd:   -1,
			wantValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			it := ParseInstruction([]byte(tt.line))
			flagToken := it.FindFlag(tt.flag)
			start, end, value := it.FlagValue(flagToken)
			assert.Equal(t, tt.wantStart, start, "start mismatch")
			assert.Equal(t, tt.wantEnd, end, "end mismatch")
			assert.Equal(t, tt.wantValue, value, "value mismatch")
		})
	}
}

func TestInstructionTokens_TokenAfter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		line    string
		keyword string
		want    *Token
	}{
		{
			name:    "token after AS",
			line:    "FROM alpine AS builder",
			keyword: "AS",
			want:    &Token{Type: TokenArgument, Value: "builder", Start: 15, End: 22},
		},
		{
			name:    "token after AS with extra spaces",
			line:    "FROM alpine AS   builder",
			keyword: "AS",
			want:    &Token{Type: TokenArgument, Value: "builder", Start: 17, End: 24},
		},
		{
			name:    "no token after",
			line:    "FROM alpine AS",
			keyword: "AS",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			it := ParseInstruction([]byte(tt.line))
			keyword := it.FindKeyword(tt.keyword)
			require.NotNil(t, keyword)
			got := it.TokenAfter(keyword)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTokenizerQuotedStrings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		line string
		// We just check that it doesn't panic and parses reasonably
	}{
		{name: "double quoted arg", line: `COPY "file with spaces" /dest`},
		{name: "single quoted arg", line: `COPY 'file with spaces' /dest`},
		{name: "escaped quote", line: `COPY "file\"name" /dest`},
		{name: "escaped in single", line: `COPY 'file\'name' /dest`},
		{name: "mixed quotes", line: `COPY "it's fine" /dest`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tokens := TokenizeLine([]byte(tt.line))
			assert.NotEmpty(t, tokens)
			// Verify all byte ranges are valid and non-overlapping
			for i := 1; i < len(tokens); i++ {
				assert.GreaterOrEqual(t, tokens[i].Start, tokens[i-1].End,
					"tokens should not overlap")
			}
		})
	}
}

func TestTokenizerEdgeCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		line string
	}{
		{name: "empty line", line: ""},
		{name: "only whitespace", line: "   "},
		{name: "only keyword", line: "FROM"},
		{name: "unclosed quote", line: `COPY "unclosed`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Should not panic
			tokens := TokenizeLine([]byte(tt.line))
			// All tokens should have valid ranges
			for _, tok := range tokens {
				assert.LessOrEqual(t, tok.Start, tok.End)
				assert.LessOrEqual(t, tok.End, len(tt.line))
			}
		})
	}
}

// TestByteOffsetAccuracy verifies that token positions correctly index into source.
func TestByteOffsetAccuracy(t *testing.T) {
	t.Parallel()
	line := "FROM alpine:3.18 AS builder"
	lineBytes := []byte(line)
	tokens := TokenizeLine(lineBytes)

	for _, tok := range tokens {
		// Extract using byte offsets
		extracted := string(lineBytes[tok.Start:tok.End])
		assert.Equal(t, tok.Value, extracted,
			"token value should match extracted bytes for %q", tok.Value)
	}
}
