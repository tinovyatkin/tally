package lspserver

import (
	"github.com/sourcegraph/jsonrpc2"

	protocol "github.com/tinovyatkin/tally/internal/lsp/protocol"
)

const commandApplyAllFixes = "tally.applyAllFixes"

// handleExecuteCommand dispatches workspace/executeCommand requests.
func (s *Server) handleExecuteCommand(params *protocol.ExecuteCommandParams) (any, error) {
	switch params.Command {
	case commandApplyAllFixes:
		return s.executeApplyAllFixes(params)
	default:
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "unknown command: " + params.Command,
		}
	}
}

// executeApplyAllFixes applies all safe fixes to the document specified in Arguments[0].
func (s *Server) executeApplyAllFixes(params *protocol.ExecuteCommandParams) (any, error) {
	if params.Arguments == nil || len(*params.Arguments) == 0 {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "tally.applyAllFixes requires a document URI argument",
		}
	}

	uri, ok := (*params.Arguments)[0].(string)
	if !ok {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "tally.applyAllFixes: argument must be a document URI string",
		}
	}

	doc := s.documents.Get(uri)
	if doc == nil {
		return nil, nil //nolint:nilnil // LSP: null result is valid for "no edits"
	}

	edits := s.computeSafeFixes(doc)
	if len(edits) == 0 {
		return nil, nil //nolint:nilnil // no changes
	}

	return &protocol.WorkspaceEdit{
		Changes: ptrTo(map[protocol.DocumentUri][]*protocol.TextEdit{
			protocol.DocumentUri(uri): edits,
		}),
	}, nil
}
