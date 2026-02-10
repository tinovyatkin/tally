package lspserver

import (
	"context"
	"path/filepath"

	protocol "github.com/tinovyatkin/tally/internal/lsp/protocol"

	"github.com/tinovyatkin/tally/internal/config"
	"github.com/tinovyatkin/tally/internal/fix"
	"github.com/tinovyatkin/tally/internal/linter"
	"github.com/tinovyatkin/tally/internal/processor"
	"github.com/tinovyatkin/tally/internal/rules"
)

// computeSafeFixes runs the full lint+fix pipeline and returns LSP TextEdits
// for all safe auto-fixes applicable to the document. Returns nil if no fixes.
func (s *Server) computeSafeFixes(doc *Document) []*protocol.TextEdit {
	content := []byte(doc.Content)
	input := s.lintInput(doc.URI, content)

	// 1. Lint + filter: reuse shared pipeline.
	result, err := linter.LintFile(input)
	if err != nil {
		return nil
	}

	chain := linter.LSPProcessors()
	procCtx := processor.NewContext(
		map[string]*config.Config{input.FilePath: result.Config},
		result.Config,
		map[string][]byte{input.FilePath: content},
	)
	violations := chain.Process(result.Violations, procCtx)

	// 2. Apply style-safe fixes via existing fix infrastructure.
	// The fixer handles conflict resolution and ordering.
	fixer := &fix.Fixer{SafetyThreshold: fix.FixSafe}
	fixResult, err := fixer.Apply(context.Background(), violations, map[string][]byte{input.FilePath: content})
	if err != nil {
		return nil
	}

	// 3. Collect original edits from applied fixes and convert to LSP TextEdits.
	// The fixer records the original (pre-adjustment) edits in AppliedFix,
	// which reference positions in the original document — exactly what LSP needs.
	change := fixResult.Changes[filepath.Clean(input.FilePath)]
	if change == nil || !change.HasChanges() {
		return nil
	}

	var allEdits []rules.TextEdit
	for _, af := range change.FixesApplied {
		allEdits = append(allEdits, af.Edits...)
	}
	return convertTextEdits(allEdits)
}

// handleFormatting handles textDocument/formatting by applying safe auto-fixes.
func (s *Server) handleFormatting(params *protocol.DocumentFormattingParams) (any, error) {
	doc := s.documents.Get(string(params.TextDocument.Uri))
	if doc == nil {
		return nil, nil //nolint:nilnil // LSP: null result is valid for "no edits"
	}

	edits := s.computeSafeFixes(doc)
	if len(edits) == 0 {
		return nil, nil //nolint:nilnil // LSP: null result is valid for "no edits"
	}
	return edits, nil
}
