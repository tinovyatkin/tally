# Repository Guidelines

## Project Structure & Module Organization

This repository is a Go CLI for linting Dockerfiles and Containerfiles. It checks container build files for best practices, security issues, and common mistakes.

- `main.go`: application entrypoint
- `cmd/tally/cmd/`: CLI commands (`root.go`, `check.go`, `version.go`)
- `internal/`: implementation packages (Dockerfile parsing, linting rules, version)
- `internal/integration/`: end-to-end tests with snapshots and fixtures
  - `internal/integration/testdata/<case>/Dockerfile`: test Dockerfiles
  - `internal/integration/__snapshots__/`: `go-snaps` snapshot outputs
- `bin/` and `dist/`: local tools / release artifacts (ignored by Git)

## Build, Test, and Development Commands

- `make build`: builds the `tally` binary into the repo root
- `make test`: runs `go test -race -count=1 -timeout=30s ./...`
- `make lint`: runs `golangci-lint` for CI (no auto-fix)
- `make lint-fix`: runs `golangci-lint` with `--fix` for local development
- `make clean`: removes the built binary and deletes `bin/` + `dist/`

Local usage examples:
- `go run . check --help`
- `go run . check Dockerfile`
- `go run . check --max-lines 100 Dockerfile`

## Coding Style & Naming Conventions

- Format: `gofmt` + `goimports` (configured via `.golangci.yaml`, with `github.com/tinovyatkin/tally` as the local import prefix).
- Prefer small, focused packages under `internal/`; keep CLI wiring in `cmd/`.
- Tests use standard Go conventions: filenames end in `*_test.go`.

## Testing Guidelines

- Unit tests live alongside packages in `internal/**`.
- Integration tests (`internal/integration`) build the binary once and run it against test fixtures.
- Update snapshots when intentional output changes:
  - `UPDATE_SNAPS=true go test ./internal/integration/...`

## Commit & Pull Request Guidelines

- Follow semantic commit rules (Conventional Commits), e.g. `feat: ...`, `fix: ...`, `chore: ...` (enforced via `commitlint` in `.lefthook.yml`).
- Run `make lint` and `make test` before opening a PR (Lefthook runs these on `pre-commit` and `make build` on `pre-push`).
- PRs should explain *what* changed and *why*, note any snapshot updates, and avoid committing build outputs (the `tally` binary is Git-ignored).
