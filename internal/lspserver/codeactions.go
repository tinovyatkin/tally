package lspserver

import (
	"go.lsp.dev/protocol"

	"github.com/tinovyatkin/tally/internal/rules"
)

// codeActionsForDocument returns quick-fix code actions for the given range.
func (s *Server) codeActionsForDocument(
	doc *Document,
	params *protocol.CodeActionParams,
) []protocol.CodeAction {
	filePath := uriToPath(doc.URI)
	content := []byte(doc.Content)

	violations := lintFile(filePath, content)

	actions := make([]protocol.CodeAction, 0, len(violations))

	for _, v := range violations {
		if v.SuggestedFix == nil || v.SuggestedFix.NeedsResolve {
			continue
		}
		if len(v.SuggestedFix.Edits) == 0 {
			continue
		}

		vRange := violationRange(v)
		if !rangesOverlap(vRange, params.Range) {
			continue
		}

		edits := convertTextEdits(v.SuggestedFix.Edits)
		if len(edits) == 0 {
			continue
		}

		kind := protocol.QuickFix
		isPreferred := v.SuggestedFix.IsPreferred || v.SuggestedFix.Safety == rules.FixSafe
		action := protocol.CodeAction{
			Title:       v.SuggestedFix.Description,
			Kind:        kind,
			IsPreferred: isPreferred,
			Diagnostics: matchingDiagnostics(v, params.Context.Diagnostics),
			Edit: &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentURI][]protocol.TextEdit{
					params.TextDocument.URI: edits,
				},
			},
		}
		actions = append(actions, action)
	}

	return actions
}

// convertTextEdits converts tally TextEdits to LSP TextEdits.
func convertTextEdits(edits []rules.TextEdit) []protocol.TextEdit {
	result := make([]protocol.TextEdit, 0, len(edits))
	for _, e := range edits {
		loc := e.Location
		if loc.IsFileLevel() {
			continue
		}

		startLine := clampUint32(loc.Start.Line - 1)
		startChar := clampUint32(loc.Start.Column)
		endLine := startLine
		endChar := startChar

		if !loc.IsPointLocation() {
			endLine = clampUint32(loc.End.Line - 1)
			endChar = clampUint32(loc.End.Column)
		}

		result = append(result, protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{Line: startLine, Character: startChar},
				End:   protocol.Position{Line: endLine, Character: endChar},
			},
			NewText: e.NewText,
		})
	}
	return result
}

// rangesOverlap checks if two LSP ranges overlap.
func rangesOverlap(a, b protocol.Range) bool {
	if a.End.Line < b.Start.Line || (a.End.Line == b.Start.Line && a.End.Character < b.Start.Character) {
		return false
	}
	if b.End.Line < a.Start.Line || (b.End.Line == a.Start.Line && b.End.Character < a.Start.Character) {
		return false
	}
	return true
}

// matchingDiagnostics finds diagnostics that match a violation by line and code.
func matchingDiagnostics(v rules.Violation, diagnostics []protocol.Diagnostic) []protocol.Diagnostic {
	vRange := violationRange(v)
	var matched []protocol.Diagnostic
	for _, d := range diagnostics {
		if d.Range.Start.Line == vRange.Start.Line {
			if code, ok := d.Code.(string); ok && code == v.RuleCode {
				matched = append(matched, d)
			}
		}
	}
	return matched
}
