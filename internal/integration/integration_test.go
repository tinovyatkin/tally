package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
)

var (
	binaryPath  string
	coverageDir string
)

func TestMain(m *testing.M) {
	// Build the binary once before running tests
	tmpDir, err := os.MkdirTemp("", "tally-test")
	if err != nil {
		panic(err)
	}

	binaryPath = filepath.Join(tmpDir, "tally")

	// Create coverage directory in project root for persistent coverage data
	// If GOCOVERDIR is set externally, use that; otherwise use "./coverage"
	coverageDir = os.Getenv("GOCOVERDIR")
	if coverageDir == "" {
		// Get absolute path to project root (2 levels up from internal/integration)
		wd, err := os.Getwd()
		if err != nil {
			_ = os.RemoveAll(tmpDir)
			panic("failed to get working directory: " + err.Error())
		}
		coverageDir = filepath.Join(wd, "..", "..", "coverage")
	}
	// Make path absolute
	coverageDir, err = filepath.Abs(coverageDir)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		panic("failed to get absolute coverage directory path: " + err.Error())
	}
	if err := os.MkdirAll(coverageDir, 0o750); err != nil {
		_ = os.RemoveAll(tmpDir)
		panic("failed to create coverage directory: " + err.Error())
	}

	// Build the module's main package with coverage instrumentation
	cmd := exec.Command("go", "build", "-cover", "-o", binaryPath, "github.com/tinovyatkin/tally")
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.RemoveAll(tmpDir)
		panic("failed to build binary: " + string(out))
	}

	code := m.Run()

	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}

func TestCheck(t *testing.T) {
	testCases := []struct {
		name     string
		dir      string
		args     []string
		wantExit int
	}{
		{"simple", "simple", []string{"--format", "json"}, 0},
		{"simple-max-lines-pass", "simple", []string{"--max-lines", "100", "--format", "json"}, 0},
		{"simple-max-lines-fail", "simple", []string{"--max-lines", "2", "--format", "json"}, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dockerfilePath := filepath.Join("testdata", tc.dir, "Dockerfile")

			args := append([]string{"check"}, tc.args...)
			args = append(args, dockerfilePath)
			cmd := exec.Command(binaryPath, args...)
			cmd.Env = append(os.Environ(),
				"GOCOVERDIR="+coverageDir,
			)
			output, err := cmd.CombinedOutput()

			// Check exit code if expected to fail
			if tc.wantExit != 0 {
				if err == nil {
					t.Errorf("expected exit code %d, got 0", tc.wantExit)
				}
			}

			snaps.WithConfig(snaps.Ext(".json")).MatchStandaloneSnapshot(t, string(output))
		})
	}
}

func TestVersion(t *testing.T) {
	cmd := exec.Command(binaryPath, "version")
	cmd.Env = append(os.Environ(),
		"GOCOVERDIR="+coverageDir,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("version command failed: %v\noutput: %s", err, output)
	}

	// Version output contains "dev" in tests
	if len(output) == 0 {
		t.Error("expected version output, got empty")
	}
}
