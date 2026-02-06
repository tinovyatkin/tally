// Package lsptest implements black-box protocol tests for the tally LSP server.
//
// Each test launches tally lsp --stdio as a real subprocess and communicates
// over Content-Length-framed JSON-RPC on stdin/stdout. Coverage data from the
// subprocess is collected via GOCOVERDIR (same mechanism as internal/integration/).
package lsptest

import (
	"context"
	"testing"
	"time"

	"github.com/gkampitakis/go-snaps/match"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestLSP_Initialize(t *testing.T) {
	t.Parallel()
	ts := startTestServer(t)
	result := ts.initialize(t)

	// Snapshot the full server capabilities; version is dynamic.
	snaps.MatchStandaloneJSON(t, result, match.Any("serverInfo.version"))
}

func TestLSP_ShutdownExit(t *testing.T) {
	t.Parallel()
	ts := startTestServer(t)
	ts.initialize(t)

	// Shutdown should succeed without error.
	ts.shutdown(t)

	// After exit notification, the subprocess should terminate.
	exited := make(chan error, 1)
	go func() { exited <- ts.cmd.Wait() }()

	select {
	case <-exited:
		// Process exited (exit code may be non-zero due to jsonrpc2 handler teardown).
	case <-time.After(5 * time.Second):
		t.Fatal("server process did not exit after shutdown+exit")
	}
}

func TestLSP_DiagnosticsOnDidOpen(t *testing.T) {
	t.Parallel()
	ts := startTestServer(t)
	ts.initialize(t)

	uri := protocol.DocumentURI("file:///tmp/test-didopen/Dockerfile")
	ts.openDocument(t, uri, "FROM alpine:3.18\nMAINTAINER test@example.com\n")

	diag := ts.waitDiagnostics(t)

	// Snapshot the full diagnostics response.
	snaps.MatchStandaloneJSON(t, diag)
}

func TestLSP_DiagnosticsUpdatedOnDidChange(t *testing.T) {
	t.Parallel()
	ts := startTestServer(t)
	ts.initialize(t)

	uri := protocol.DocumentURI("file:///tmp/test-didchange/Dockerfile")

	// Open with MAINTAINER → expect diagnostics.
	ts.openDocument(t, uri, "FROM alpine:3.18\nMAINTAINER test@example.com\n")
	diag1 := ts.waitDiagnostics(t)
	require.NotEmpty(t, diag1.Diagnostics)

	hasMaintainer := func(diags []protocol.Diagnostic) bool {
		for _, d := range diags {
			if code, ok := d.Code.(string); ok && code == "buildkit/MaintainerDeprecated" {
				return true
			}
		}
		return false
	}
	assert.True(t, hasMaintainer(diag1.Diagnostics), "expected MaintainerDeprecated after open")

	// Change: remove MAINTAINER → diagnostics should no longer include it.
	ts.changeDocument(t, uri, 2, "FROM alpine:3.18\nLABEL maintainer=\"test@example.com\"\n")
	diag2 := ts.waitDiagnostics(t)
	assert.False(t, hasMaintainer(diag2.Diagnostics), "MaintainerDeprecated should be gone after change")
}

func TestLSP_DiagnosticsClearedOnClose(t *testing.T) {
	t.Parallel()
	ts := startTestServer(t)
	ts.initialize(t)

	uri := protocol.DocumentURI("file:///tmp/test-didclose/Dockerfile")

	ts.openDocument(t, uri, "FROM alpine:3.18\nMAINTAINER test@example.com\n")
	diag1 := ts.waitDiagnostics(t)
	require.NotEmpty(t, diag1.Diagnostics)

	// Close the document → server should publish empty diagnostics.
	ts.closeDocument(t, uri)
	diag2 := ts.waitDiagnostics(t)
	assert.Equal(t, uri, diag2.URI)
	assert.Empty(t, diag2.Diagnostics, "expected empty diagnostics after close")
}

func TestLSP_DiagnosticsOnDidSave(t *testing.T) {
	t.Parallel()
	ts := startTestServer(t)
	ts.initialize(t)

	uri := protocol.DocumentURI("file:///tmp/test-didsave/Dockerfile")

	// Open a clean file.
	ts.openDocument(t, uri, "FROM alpine:3.18\nRUN echo hello\n")
	diag1 := ts.waitDiagnostics(t)

	hasMaintainer := func(diags []protocol.Diagnostic) bool {
		for _, d := range diags {
			if code, ok := d.Code.(string); ok && code == "buildkit/MaintainerDeprecated" {
				return true
			}
		}
		return false
	}
	assert.False(t, hasMaintainer(diag1.Diagnostics))

	// Save with new text that includes MAINTAINER.
	ts.saveDocument(t, uri, "FROM alpine:3.18\nMAINTAINER test@example.com\n")
	diag2 := ts.waitDiagnostics(t)
	assert.True(t, hasMaintainer(diag2.Diagnostics), "expected MaintainerDeprecated after save")
}

func TestLSP_CodeAction(t *testing.T) {
	t.Parallel()
	ts := startTestServer(t)
	ts.initialize(t)

	uri := protocol.DocumentURI("file:///tmp/test-codeaction/Dockerfile")
	ts.openDocument(t, uri, "FROM alpine:3.18\nMAINTAINER test@example.com\n")

	diag := ts.waitDiagnostics(t)
	require.NotEmpty(t, diag.Diagnostics)

	// Find the MaintainerDeprecated diagnostic.
	var maintainerDiag *protocol.Diagnostic
	for i, d := range diag.Diagnostics {
		if code, ok := d.Code.(string); ok && code == "buildkit/MaintainerDeprecated" {
			maintainerDiag = &diag.Diagnostics[i]
			break
		}
	}
	require.NotNil(t, maintainerDiag, "expected MaintainerDeprecated diagnostic for code action test")

	// Request code actions for the MAINTAINER line.
	ctx, cancel := context.WithTimeout(context.Background(), diagTimeout)
	defer cancel()

	var actions []protocol.CodeAction
	_, err := ts.conn.Call(ctx, protocol.MethodTextDocumentCodeAction, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range:        maintainerDiag.Range,
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{*maintainerDiag},
		},
	}, &actions)
	require.NoError(t, err)

	// Snapshot the full code actions response.
	snaps.MatchStandaloneJSON(t, actions)
}

func TestLSP_MethodNotFound(t *testing.T) {
	t.Parallel()
	ts := startTestServer(t)
	ts.initialize(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := ts.conn.Call(ctx, "custom/nonExistentMethod", nil, nil)
	assert.Error(t, err, "unknown method should return an error")
}
