# CLAUDE.md - Project Guidance

## Project Overview

`tally` is a fast, configurable linter for Dockerfiles and Containerfiles. It checks container build files for best practices, security issues, and common mistakes.

## Design Philosophy

**Minimize code ownership** - This project heavily reuses existing, well-maintained libraries:
- `github.com/moby/buildkit/frontend/dockerfile/parser` - Official Dockerfile parsing
- `github.com/urfave/cli/v3` - CLI framework
- `golang.org/x/sync` - Concurrency primitives

Do not re-implement functionality that exists in these libraries.

**Adding dependencies** - Before adding a new dependency, run `go list -m -versions <module>` to check available versions and use the latest stable release.

## Build & Test Commands

```bash
# Build
go build ./...

# Run all tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Update snapshots for integration tests
UPDATE_SNAPS=true go test ./internal/integration/...

# Run the CLI
go run . check --help
go run . check Dockerfile
go run . check --max-lines 100 Dockerfile
```

## Coverage Collection

Integration tests are built with coverage instrumentation (`-cover` flag). Coverage data is automatically collected to a temporary directory during test runs.

```bash
# Run integration tests (coverage data is automatically collected)
go test ./internal/integration/...

# To view coverage reports, manually run with a persistent coverage directory:
# 1. Build the binary with coverage
go build -cover -o tally-cover .

# 2. Run tests with GOCOVERDIR set
mkdir coverage
GOCOVERDIR=coverage go test ./internal/integration/...

# 3. Generate coverage reports
go tool covdata percent -i=coverage
go tool covdata textfmt -i=coverage -o=coverage.txt
go tool cover -html=coverage.txt -o=coverage.html
```

## Commit Messages

- Use semantic commit rules (Conventional Commits), e.g. `feat: ...`, `fix: ...`, `chore: ...` (enforced via `commitlint` in `.lefthook.yml`).

## Project Structure

```
.
├── main.go                           # Entry point
├── cmd/tally/cmd/                    # CLI commands (urfave/cli)
│   ├── root.go                       # Root command setup
│   ├── check.go                      # Check subcommand (linting)
│   └── version.go                    # Version subcommand
├── internal/
│   ├── dockerfile/                   # Dockerfile parsing (uses buildkit)
│   │   ├── parser.go
│   │   └── parser_test.go
│   ├── lint/                         # Linting rules
│   │   ├── rules.go
│   │   └── rules_test.go
│   ├── version/
│   │   └── version.go
│   ├── integration/                  # Integration tests (go-snaps)
│   │   ├── integration_test.go
│   │   ├── __snapshots__/
│   │   └── testdata/                 # Test fixtures (each in own directory)
│   └── testutil/                     # Test utilities
├── packaging/
│   ├── pack.rb                       # Packaging orchestration script
│   ├── npm/                          # npm package structure (@contino/tally)
│   ├── pypi/                         # Python package (tally-cli)
│   └── rubygems/                     # Ruby gem (tally-cli)
└── README.md
```

## Testing Strategy

**Integration tests are the preferred way to test and develop new features.** They provide true end-to-end coverage, ensuring the entire pipeline works correctly.

### Integration Tests (`internal/integration/`)

**How it works:**
1. `TestMain` builds the CLI binary with `-cover` flag for coverage instrumentation
2. Tests run the CLI binary against test Dockerfiles
3. Snapshots (`go-snaps`) verify the JSON output

**Adding a new test case:**
1. Create a new directory under `internal/integration/testdata/` with a `Dockerfile`
2. Add a test case to `TestCheck`
3. Run `UPDATE_SNAPS=true go test ./internal/integration/...` to generate snapshots

### Unit Tests

- Standard Go tests for isolated parsing and linting logic
- Use when testing pure functions that don't require CLI interaction

### Test Fixtures

Test fixtures are organized in separate directories under `testdata/` to support future context-aware features (dockerignore, config files, etc.)

## Key Flags

- `--max-lines, -l`: Maximum number of lines allowed (0 = unlimited)
- `--format, -f`: Output format (text, json)

## Adding New Linting Rules

1. Add the rule logic to `internal/lint/rules.go`
2. Add unit tests to `internal/lint/rules_test.go`
3. Add CLI flag to `cmd/tally/cmd/check.go`
4. Add integration test cases to `internal/integration/`
5. Update documentation

## Package Publishing

Published to three package managers:
- **NPM**: `@contino/tally` (with platform-specific optional dependencies)
- **PyPI**: `tally-cli`
- **RubyGems**: `tally-cli`

Publishing is handled by the `packaging/pack.rb` script and GitHub Actions.
