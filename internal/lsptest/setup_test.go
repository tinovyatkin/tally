package lsptest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

var (
	binaryPath  string
	coverageDir string
)

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "tally-lsptest")
	if err != nil {
		panic(err)
	}

	binaryName := "tally"
	if runtime.GOOS == "windows" {
		binaryName = "tally.exe"
	}
	binaryPath = filepath.Join(tmpDir, binaryName)

	// Reuse the same coverage directory as integration tests.
	coverageDir = os.Getenv("GOCOVERDIR")
	if coverageDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			_ = os.RemoveAll(tmpDir)
			panic("failed to get working directory: " + err.Error())
		}
		coverageDir = filepath.Join(wd, "..", "..", "coverage")
	}
	coverageDir, err = filepath.Abs(coverageDir)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		panic("failed to get absolute coverage directory path: " + err.Error())
	}
	if err := os.MkdirAll(coverageDir, 0o750); err != nil {
		_ = os.RemoveAll(tmpDir)
		panic("failed to create coverage directory: " + err.Error())
	}

	// Build the binary with coverage instrumentation.
	cmd := exec.Command("go", "build", "-cover", "-o", binaryPath, "github.com/tinovyatkin/tally")
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.RemoveAll(tmpDir)
		panic("failed to build binary: " + string(out))
	}

	code := m.Run()
	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}

// processIO wraps subprocess stdin/stdout as an io.ReadWriteCloser
// for use with jsonrpc2.NewStream.
type processIO struct {
	reader io.ReadCloser
	writer io.WriteCloser
}

func (p *processIO) Read(data []byte) (int, error)  { return p.reader.Read(data) }
func (p *processIO) Write(data []byte) (int, error) { return p.writer.Write(data) }
func (p *processIO) Close() error                   { return p.writer.Close() }

// testServer manages a tally lsp --stdio subprocess for black-box testing.
type testServer struct {
	cmd    *exec.Cmd
	conn   jsonrpc2.Conn
	stderr *bytes.Buffer

	diagnosticsCh chan *protocol.PublishDiagnosticsParams
}

// startTestServer launches tally lsp --stdio as a subprocess with
// Content-Length-framed JSON-RPC over stdin/stdout.
func startTestServer(t *testing.T) *testServer {
	t.Helper()

	cmd := exec.Command(binaryPath, "lsp", "--stdio")
	cmd.Env = append(os.Environ(), "GOCOVERDIR="+coverageDir)

	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	require.NoError(t, cmd.Start())

	stream := jsonrpc2.NewStream(&processIO{reader: stdout, writer: stdin})
	conn := jsonrpc2.NewConn(stream)

	ts := &testServer{
		cmd:           cmd,
		conn:          conn,
		stderr:        &stderr,
		diagnosticsCh: make(chan *protocol.PublishDiagnosticsParams, 10),
	}

	// Route server-to-client notifications.
	conn.Go(context.Background(), func(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
		if req.Method() == protocol.MethodTextDocumentPublishDiagnostics {
			var params protocol.PublishDiagnosticsParams
			if err := json.Unmarshal(req.Params(), &params); err == nil {
				ts.diagnosticsCh <- &params
			}
		}
		return reply(ctx, nil, nil)
	})

	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("server stderr:\n%s", stderr.String())
		}
		if err := conn.Close(); err != nil {
			t.Logf("lsp conn close: %v", err)
		}
		// Wait for process with timeout; kill if it doesn't exit.
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			if err := cmd.Process.Kill(); err != nil {
				t.Logf("kill lsp server: %v", err)
			}
			<-done
		}
	})

	return ts
}

// initialize sends initialize + initialized and returns the server capabilities.
func (ts *testServer) initialize(t *testing.T) protocol.InitializeResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result protocol.InitializeResult
	_, err := ts.conn.Call(ctx, protocol.MethodInitialize, &protocol.InitializeParams{
		ClientInfo: &protocol.ClientInfo{
			Name:    "tally-lsptest",
			Version: "1.0.0",
		},
	}, &result)
	require.NoError(t, err)

	require.NoError(t, ts.conn.Notify(ctx, protocol.MethodInitialized, &protocol.InitializedParams{}))

	return result
}

const diagTimeout = 10 * time.Second

// waitDiagnostics blocks until a publishDiagnostics notification arrives or timeout.
func (ts *testServer) waitDiagnostics(t *testing.T) *protocol.PublishDiagnosticsParams {
	t.Helper()
	select {
	case d := <-ts.diagnosticsCh:
		return d
	case <-time.After(diagTimeout):
		t.Fatal("timed out waiting for diagnostics")
		return nil
	}
}

// shutdown sends the shutdown request followed by exit notification.
func (ts *testServer) shutdown(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := ts.conn.Call(ctx, protocol.MethodShutdown, nil, nil)
	require.NoError(t, err)

	require.NoError(t, ts.conn.Notify(ctx, protocol.MethodExit, nil))
}

// openDocument sends textDocument/didOpen.
func (ts *testServer) openDocument(t *testing.T, uri protocol.DocumentURI, content string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, ts.conn.Notify(ctx, protocol.MethodTextDocumentDidOpen, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "dockerfile",
			Version:    1,
			Text:       content,
		},
	}))
}

// changeDocument sends textDocument/didChange with full sync.
func (ts *testServer) changeDocument(t *testing.T, uri protocol.DocumentURI, version int32, content string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, ts.conn.Notify(ctx, protocol.MethodTextDocumentDidChange, &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
			Version:                version,
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			{Text: content},
		},
	}))
}

// closeDocument sends textDocument/didClose.
func (ts *testServer) closeDocument(t *testing.T, uri protocol.DocumentURI) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, ts.conn.Notify(ctx, protocol.MethodTextDocumentDidClose, &protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	}))
}

// saveDocument sends textDocument/didSave with text included.
func (ts *testServer) saveDocument(t *testing.T, uri protocol.DocumentURI, content string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, ts.conn.Notify(ctx, protocol.MethodTextDocumentDidSave, &protocol.DidSaveTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Text:         content,
	}))
}
