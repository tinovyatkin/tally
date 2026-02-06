// Package lspserver implements a Language Server Protocol server for tally.
//
// The server provides Dockerfile linting diagnostics, quick-fix code actions,
// and document formatting through the LSP protocol. It reuses the same lint
// pipeline as the CLI (dockerfile.Parse, semantic model, rules, processors).
//
// Transport: stdio only (--stdio) for v1.
// Protocol: LSP 3.16 types via go.lsp.dev/protocol, JSON-RPC via go.lsp.dev/jsonrpc2.
package lspserver

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	"github.com/tinovyatkin/tally/internal/version"
)

const serverName = "tally"

// Server is the tally LSP server.
type Server struct {
	conn      jsonrpc2.Conn
	documents *DocumentStore
}

// New creates a new LSP server.
func New() *Server {
	return &Server{
		documents: NewDocumentStore(),
	}
}

// RunStdio starts the LSP server on stdin/stdout.
// It blocks until the connection is closed or the context is cancelled.
func (s *Server) RunStdio(ctx context.Context) error {
	stream := jsonrpc2.NewStream(stdioReadWriteCloser{})
	conn := jsonrpc2.NewConn(stream)
	s.conn = conn

	conn.Go(ctx, jsonrpc2.AsyncHandler(jsonrpc2.ReplyHandler(s.handle)))

	select {
	case <-ctx.Done():
		return conn.Close()
	case <-conn.Done():
		return conn.Err()
	}
}

// handle dispatches incoming JSON-RPC messages to the appropriate handler.
func (s *Server) handle(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	switch req.Method() {
	// Lifecycle
	case protocol.MethodInitialize:
		return s.handleInitialize(ctx, reply, req)
	case protocol.MethodInitialized:
		return reply(ctx, nil, nil)
	case protocol.MethodShutdown:
		return reply(ctx, nil, nil)
	case protocol.MethodExit:
		return s.conn.Close()
	case protocol.MethodSetTrace:
		return reply(ctx, nil, nil)

	// Document sync
	case protocol.MethodTextDocumentDidOpen:
		return s.handleDidOpen(ctx, reply, req)
	case protocol.MethodTextDocumentDidChange:
		return s.handleDidChange(ctx, reply, req)
	case protocol.MethodTextDocumentDidSave:
		return s.handleDidSave(ctx, reply, req)
	case protocol.MethodTextDocumentDidClose:
		return s.handleDidClose(ctx, reply, req)

	// Language features
	case protocol.MethodTextDocumentCodeAction:
		return s.handleCodeAction(ctx, reply, req)

	// Workspace
	case protocol.MethodWorkspaceDidChangeConfiguration:
		return reply(ctx, nil, nil)

	default:
		return jsonrpc2.MethodNotFoundHandler(ctx, reply, req)
	}
}

// handleInitialize responds to the initialize request with server capabilities.
func (s *Server) handleInitialize(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.InitializeParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return replyParseError(ctx, reply, err)
	}

	log.Printf("lsp: initialize from %s", clientInfoString(params.ClientInfo))

	syncKind := protocol.TextDocumentSyncKindFull
	ver := version.RawVersion()

	result := protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			TextDocumentSync: protocol.TextDocumentSyncOptions{
				OpenClose: true,
				Change:    syncKind,
				Save: &protocol.SaveOptions{
					IncludeText: true,
				},
			},
			CodeActionProvider: &protocol.CodeActionOptions{
				CodeActionKinds: []protocol.CodeActionKind{
					protocol.QuickFix,
					"source.fixAll.tally",
				},
			},
		},
		ServerInfo: &protocol.ServerInfo{
			Name:    serverName,
			Version: ver,
		},
	}

	return reply(ctx, result, nil)
}

// handleDidOpen handles textDocument/didOpen by linting the opened document.
func (s *Server) handleDidOpen(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.DidOpenTextDocumentParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return replyParseError(ctx, reply, err)
	}

	uri := string(params.TextDocument.URI)
	s.documents.Open(uri, string(params.TextDocument.LanguageID), params.TextDocument.Version, params.TextDocument.Text)

	if doc := s.documents.Get(uri); doc != nil {
		s.publishDiagnostics(ctx, doc)
	}
	return reply(ctx, nil, nil)
}

// handleDidChange handles textDocument/didChange by updating the document and re-linting.
func (s *Server) handleDidChange(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.DidChangeTextDocumentParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return replyParseError(ctx, reply, err)
	}

	uri := string(params.TextDocument.URI)

	// With full sync, there's exactly one content change containing the full text.
	for _, change := range params.ContentChanges {
		s.documents.Update(uri, params.TextDocument.Version, change.Text)
	}

	if doc := s.documents.Get(uri); doc != nil {
		s.publishDiagnostics(ctx, doc)
	}
	return reply(ctx, nil, nil)
}

// handleDidSave handles textDocument/didSave by re-linting.
func (s *Server) handleDidSave(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.DidSaveTextDocumentParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return replyParseError(ctx, reply, err)
	}

	uri := string(params.TextDocument.URI)
	if params.Text != "" {
		s.documents.Update(uri, 0, params.Text)
	}

	if doc := s.documents.Get(uri); doc != nil {
		s.publishDiagnostics(ctx, doc)
	}
	return reply(ctx, nil, nil)
}

// handleDidClose handles textDocument/didClose by clearing diagnostics and removing the document.
func (s *Server) handleDidClose(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.DidCloseTextDocumentParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return replyParseError(ctx, reply, err)
	}

	uri := string(params.TextDocument.URI)
	s.documents.Close(uri)
	s.clearDiagnostics(ctx, uri)
	return reply(ctx, nil, nil)
}

// handleCodeAction handles textDocument/codeAction by returning quick-fix actions.
func (s *Server) handleCodeAction(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.CodeActionParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return replyParseError(ctx, reply, err)
	}

	uri := string(params.TextDocument.URI)
	doc := s.documents.Get(uri)
	if doc == nil {
		return reply(ctx, nil, nil)
	}

	actions := s.codeActionsForDocument(doc, &params)
	if len(actions) == 0 {
		return reply(ctx, nil, nil)
	}
	return reply(ctx, actions, nil)
}

// replyParseError sends a JSON-RPC parse error.
func replyParseError(ctx context.Context, reply jsonrpc2.Replier, err error) error {
	return reply(ctx, nil, jsonrpc2.Errorf(jsonrpc2.ParseError, "invalid params: %v", err))
}

// clientInfoString formats client info for logging.
func clientInfoString(info *protocol.ClientInfo) string {
	if info == nil {
		return "unknown"
	}
	if info.Version != "" {
		return info.Name + " " + info.Version
	}
	return info.Name
}

// stdioReadWriteCloser wraps stdin/stdout as an io.ReadWriteCloser for JSON-RPC.
type stdioReadWriteCloser struct{}

func (stdioReadWriteCloser) Read(p []byte) (int, error)  { return os.Stdin.Read(p) }
func (stdioReadWriteCloser) Write(p []byte) (int, error) { return os.Stdout.Write(p) }
func (stdioReadWriteCloser) Close() error                { return nil }
