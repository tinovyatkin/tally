package lspserver

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	"github.com/tinovyatkin/tally/internal/rules"
)

// testPipe creates an in-memory connected pair of jsonrpc2 connections.
// Returns (clientConn, serverConn).
func testPipe(t *testing.T) (jsonrpc2.Conn, jsonrpc2.Conn) {
	t.Helper()

	// Two pipes: one for each direction.
	// client writes -> server reads (c2s)
	// server writes -> client reads (s2c)
	c2s := newPipeEnd()
	s2c := newPipeEnd()

	clientStream := jsonrpc2.NewStream(rwc{reader: s2c, writer: c2s})
	serverStream := jsonrpc2.NewStream(rwc{reader: c2s, writer: s2c})

	clientConn := jsonrpc2.NewConn(clientStream)
	serverConn := jsonrpc2.NewConn(serverStream)

	t.Cleanup(func() {
		_ = clientConn.Close()
		_ = serverConn.Close()
	})

	return clientConn, serverConn
}

func TestInitializeHandshake(t *testing.T) {
	ctx := context.Background()
	clientConn, serverConn := testPipe(t)

	s := New()
	s.conn = serverConn
	serverConn.Go(ctx, jsonrpc2.AsyncHandler(jsonrpc2.ReplyHandler(s.handle)))
	clientConn.Go(ctx, jsonrpc2.MethodNotFoundHandler)

	var result protocol.InitializeResult
	_, err := clientConn.Call(ctx, protocol.MethodInitialize, &protocol.InitializeParams{
		ClientInfo: &protocol.ClientInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
	}, &result)
	require.NoError(t, err)

	assert.Equal(t, serverName, result.ServerInfo.Name)
	assert.NotEmpty(t, result.ServerInfo.Version)
}

func TestDiagnosticsOnOpen(t *testing.T) {
	ctx := t.Context()

	clientConn, serverConn := testPipe(t)

	s := New()
	s.conn = serverConn
	serverConn.Go(ctx, jsonrpc2.AsyncHandler(jsonrpc2.ReplyHandler(s.handle)))

	// Collect diagnostics notifications from the server.
	diagnosticsCh := make(chan *protocol.PublishDiagnosticsParams, 1)
	clientConn.Go(ctx, func(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
		if req.Method() == protocol.MethodTextDocumentPublishDiagnostics {
			var params protocol.PublishDiagnosticsParams
			if err := json.Unmarshal(req.Params(), &params); err == nil {
				diagnosticsCh <- &params
			}
			return reply(ctx, nil, nil)
		}
		return jsonrpc2.MethodNotFoundHandler(ctx, reply, req)
	})

	// Initialize first.
	var initResult protocol.InitializeResult
	_, err := clientConn.Call(ctx, protocol.MethodInitialize, &protocol.InitializeParams{}, &initResult)
	require.NoError(t, err)

	// Open a Dockerfile with a known issue (deprecated MAINTAINER).
	err = clientConn.Notify(ctx, protocol.MethodTextDocumentDidOpen, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        "file:///tmp/Dockerfile",
			LanguageID: "dockerfile",
			Version:    1,
			Text:       "FROM alpine:3.18\nMAINTAINER test@example.com\n",
		},
	})
	require.NoError(t, err)

	// Wait for diagnostics.
	select {
	case diag := <-diagnosticsCh:
		assert.Equal(t, protocol.DocumentURI("file:///tmp/Dockerfile"), diag.URI)
		assert.NotEmpty(t, diag.Diagnostics, "expected at least one diagnostic for deprecated MAINTAINER")
		// Check that at least one diagnostic is from tally.
		found := false
		for _, d := range diag.Diagnostics {
			if d.Source == "tally" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected diagnostics from tally source")
	case <-ctx.Done():
		t.Fatal("timed out waiting for diagnostics")
	}
}

func TestDiagnosticsClearedOnClose(t *testing.T) {
	ctx := t.Context()

	clientConn, serverConn := testPipe(t)

	s := New()
	s.conn = serverConn
	serverConn.Go(ctx, jsonrpc2.AsyncHandler(jsonrpc2.ReplyHandler(s.handle)))

	diagnosticsCh := make(chan *protocol.PublishDiagnosticsParams, 2)
	clientConn.Go(ctx, func(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
		if req.Method() == protocol.MethodTextDocumentPublishDiagnostics {
			var params protocol.PublishDiagnosticsParams
			if err := json.Unmarshal(req.Params(), &params); err == nil {
				diagnosticsCh <- &params
			}
			return reply(ctx, nil, nil)
		}
		return jsonrpc2.MethodNotFoundHandler(ctx, reply, req)
	})

	var initResult protocol.InitializeResult
	_, err := clientConn.Call(ctx, protocol.MethodInitialize, &protocol.InitializeParams{}, &initResult)
	require.NoError(t, err)

	uri := protocol.DocumentURI("file:///tmp/Dockerfile")

	// Open a file to generate diagnostics.
	err = clientConn.Notify(ctx, protocol.MethodTextDocumentDidOpen, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "dockerfile",
			Version:    1,
			Text:       "FROM alpine:3.18\nMAINTAINER test@test.com\n",
		},
	})
	require.NoError(t, err)

	// Wait for initial diagnostics.
	<-diagnosticsCh

	// Close the document.
	err = clientConn.Notify(ctx, protocol.MethodTextDocumentDidClose, &protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	require.NoError(t, err)

	// Wait for clear diagnostics.
	select {
	case diag := <-diagnosticsCh:
		assert.Equal(t, uri, diag.URI)
		assert.Empty(t, diag.Diagnostics, "expected empty diagnostics after close")
	case <-ctx.Done():
		t.Fatal("timed out waiting for clear diagnostics")
	}
}

func TestViolationRangeConversion(t *testing.T) {
	tests := []struct {
		name     string
		location rules.Location
		expected protocol.Range
	}{
		{
			name:     "file-level",
			location: rules.NewFileLocation("test"),
			expected: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
		},
		{
			name:     "line 1 col 0 (point)",
			location: rules.NewLineLocation("test", 1),
			expected: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 1000},
			},
		},
		{
			name:     "range",
			location: rules.NewRangeLocation("test", 3, 5, 3, 15),
			expected: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 5},
				End:   protocol.Position{Line: 2, Character: 15},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := rules.Violation{Location: tt.location}
			got := violationRange(v)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestSeverityConversion(t *testing.T) {
	snaps.MatchStandaloneJSON(t, map[string]protocol.DiagnosticSeverity{
		"error":   severityToLSP(rules.SeverityError),
		"warning": severityToLSP(rules.SeverityWarning),
		"info":    severityToLSP(rules.SeverityInfo),
		"style":   severityToLSP(rules.SeverityStyle),
	})
}

func TestURIToPath(t *testing.T) {
	path := uriToPath("file:///tmp/Dockerfile")
	assert.Equal(t, "/tmp/Dockerfile", path)
}
