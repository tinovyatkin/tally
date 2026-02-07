package lspserver

import (
	"bytes"
	"context"

	protocol "github.com/tinovyatkin/tally/internal/lsp/protocol"

	"github.com/tinovyatkin/tally/internal/config"
	"github.com/tinovyatkin/tally/internal/fix"
	"github.com/tinovyatkin/tally/internal/linter"
	"github.com/tinovyatkin/tally/internal/processor"
)

// handleFormatting handles textDocument/formatting by applying safe auto-fixes.
func (s *Server) handleFormatting(params *protocol.DocumentFormattingParams) (any, error) {
	doc := s.documents.Get(string(params.TextDocument.Uri))
	if doc == nil {
		return nil, nil //nolint:nilnil // LSP: null result is valid for "no edits"
	}

	content := []byte(doc.Content)
	input := s.lintInput(doc.URI, content)

	// 1. Lint + filter: reuse shared pipeline.
	result, err := linter.LintFile(input)
	if err != nil {
		return nil, nil //nolint:nilnil,nilerr // gracefully return no edits on lint error
	}

	chain := linter.LSPProcessors()
	procCtx := processor.NewContext(
		map[string]*config.Config{input.FilePath: result.Config},
		result.Config,
		map[string][]byte{input.FilePath: content},
	)
	violations := chain.Process(result.Violations, procCtx)

	// 2. Apply style-safe fixes via existing fix infrastructure.
	fixer := &fix.Fixer{SafetyThreshold: fix.FixSafe}
	fixResult, err := fixer.Apply(context.Background(), violations, map[string][]byte{input.FilePath: content})
	if err != nil {
		return nil, nil //nolint:nilnil,nilerr // gracefully return no edits on fix error
	}

	// 3. Convert to LSP text edits.
	change := fixResult.Changes[input.FilePath]
	if change == nil || !change.HasChanges() || bytes.Equal(change.OriginalContent, change.ModifiedContent) {
		return nil, nil //nolint:nilnil // no changes
	}
	return computeTextEdits(string(change.OriginalContent), string(change.ModifiedContent)), nil
}

// computeTextEdits produces a single whole-document replacement edit.
// A minimal-diff implementation can be added later for smaller edits.
func computeTextEdits(original, modified string) []*protocol.TextEdit {
	// Count lines in original to build the replacement range.
	lines := uint32(0)
	lastLineLen := uint32(0)
	for i := range len(original) {
		if original[i] == '\n' {
			lines++
			lastLineLen = 0
		} else {
			lastLineLen++
		}
	}

	return []*protocol.TextEdit{{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: lines, Character: lastLineLen},
		},
		NewText: modified,
	}}
}
