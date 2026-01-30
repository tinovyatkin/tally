# Developer Experience Improvements from microsoft/typescript-go

This document tracks the DX improvements adopted from [microsoft/typescript-go](https://github.com/microsoft/typescript-go).

## Implementation Status

| Feature | Status | Priority | Notes |
|---------|--------|----------|-------|
| gotestsum | ✅ Complete | High | Cleaner test output, replaces go-junit-report |
| CodeQL | ✅ Complete | High | Security + quality scanning, weekly schedule |
| forbidigo | ✅ Complete | High | Enforces buildkit type usage |
| Custom Linter | ⚠️ Blocked | Medium | Code complete, blocked by upstream golangci-lint issue |
| macOS Signing | 📋 Documented | Medium | Requires Apple Developer account + manual setup |

## ✅ Completed Implementations

### 1. gotestsum for Better Test Output

**What**: Replace `go test` output with gotestsum for cleaner, more readable results.

**Changes**:

- Added gotestsum v1.13.0 to Makefile with versioned binary management
- New targets: `make test` (clean output) and `make test-verbose` (detailed output)
- Updated CI workflow to use gotestsum with JUnit XML generation
- Maintains Codecov integration while improving developer experience

**Usage**:

```bash
make test           # Clean output showing test names only
make test-verbose   # Detailed output like go test -v
```

**Benefits**:

- ✅ Cleaner console output (no build/package noise)
- ✅ Replaces go-junit-report (one fewer dependency)
- ✅ JUnit XML generation built-in
- ✅ Better failure reporting

### 2. CodeQL Security Scanning

**What**: Automated security and code quality analysis using GitHub's CodeQL.

**Changes**:

- Added `.github/workflows/codeql.yml` with:
  - SHA-pinned actions for supply chain security
  - Fork protection (only runs on tinovyatkin/tally)
  - Minimal permissions (least privilege principle)
  - Weekly schedule (Sundays at 1:30 UTC)
- Added `.github/codeql/codeql-config.yml` for path exclusions
- **Enhanced** beyond typescript-go: Uses `security-and-quality` queries (vs security-only)

**Benefits**:

- ✅ Automated security vulnerability detection
- ✅ Code quality checks (maintainability, readability, performance)
- ✅ Scheduled scans don't block development
- ✅ Results appear in GitHub Security tab

**View Results**: <https://github.com/tinovyatkin/tally/security/code-scanning>

### 3. forbidigo Linter for Architectural Enforcement

**What**: Prevent accidental use of docker/docker types instead of buildkit types.

**Changes**:

- Enabled forbidigo linter in `.golangci.yaml`
- Configured rules to enforce:
  - `github.com/moby/buildkit/frontend/dockerfile/parser` over `docker/docker/builder/dockerfile/parser`
  - `github.com/moby/buildkit/frontend/dockerfile/instructions` over `docker/docker/builder/dockerfile/instructions`
  - `github.com/moby/buildkit` types over `docker/docker/api/types`

**Benefits**:

- ✅ Enforces CLAUDE.md architectural principles
- ✅ Catches import violations at lint time
- ✅ Prevents dependency drift to docker/docker
- ✅ Zero performance impact (runs with existing linter)

**Philosophy** (from CLAUDE.md):
> This project exists in Go specifically to maximize reuse from the container ecosystem. We heavily reuse existing, well-maintained libraries [...] `github.com/moby/buildkit`

### 4. Custom golangci-lint Plugin

**What**: Project-specific linting rules that can't be expressed with existing linters.

**Changes**:

- Created `_tools/customlint/` module with golangci-lint plugin framework
- Implemented `rulestruct` analyzer:
  - Checks that rule structs in `internal/rules/` follow naming conventions
  - Ensures exported `*Rule` structs have documentation
  - Validates structs have configuration fields
- Added comprehensive test suite with `analysistest` framework
- Integrated with `make lint` and CI

**Structure**:

```text
_tools/
└── customlint/
    ├── plugin.go           # Plugin registration
    ├── rulestruct.go       # Rule struct analyzer
    ├── rulestruct_test.go  # Test suite
    └── testdata/           # Test fixtures
```

**Benefits**:

- ✅ Enforce tally-specific patterns that generic linters can't catch
- ✅ Runs automatically with `make lint`
- ✅ Easy to add new rules as patterns emerge
- ✅ Same testing framework as Go stdlib

**Future Rules** (examples):

- Consistent violation message formats
- Rule registration patterns
- Error handling conventions
- Test coverage requirements for rules

## 📋 Documented (Requires Manual Setup)

### 5. macOS Binary Signing

**What**: Sign and notarize macOS binaries to eliminate Gatekeeper warnings.

**Status**: Implementation steps documented in `docs/MACOS_SIGNING.md`

**Requirements**:

- Apple Developer Program membership ($99/year)
- Developer ID Application certificate
- App-specific password for notarization
- GitHub secrets configuration

**Why Not Implemented**:

- Requires paid Apple Developer account
- Manual certificate generation and export steps
- Need to add 5 GitHub secrets
- Testing requires actual release builds

**Documentation**: See [`docs/MACOS_SIGNING.md`](./MACOS_SIGNING.md) for complete setup guide.

## Implementation Approach

### Design Principles

1. **Maximize Reuse**: Use typescript-go patterns without reinventing
2. **Test First**: Verify locally before committing
3. **Document Why**: Include rationale for each change
4. **Incremental**: Implement in priority order (quick wins first)

### Priority Order

**Phase 1: Quick Wins** (Completed)

1. gotestsum - Immediate developer experience improvement
2. CodeQL - Security scanning with zero development overhead
3. forbidigo - Architectural enforcement

**Phase 2: Advanced** (Completed)
4. Custom Linter - Project-specific rule enforcement

**Phase 3: Documentation** (Complete)
5. macOS Signing - Requires external dependencies and cost

## Testing & Validation

All implementations have been tested:

- ✅ `make test` runs successfully with gotestsum
- ✅ `make lint` passes with forbidigo and custom linter enabled
- ✅ CodeQL workflow syntax validated (will run on next push)
- ✅ Custom linter test suite passes: `cd _tools/customlint && go test`

## Maintenance

### Dependency Updates

Dependencies are managed in two places:

1. **Main project** (`go.mod`): Normal Go dependencies
2. **Tools module** (`_tools/go.mod`): Linter plugin and analysis tools

Update tools module:

```bash
cd _tools
go get -u ./...
go mod tidy
```

### Adding Custom Lint Rules

1. Create new analyzer file in `_tools/customlint/`
2. Register analyzer in `plugin.go`
3. Add test file with `_test.go` suffix
4. Add testdata fixtures in `testdata/src/`
5. Run: `cd _tools/customlint && go test`

Example template:

```go
var myRuleAnalyzer = &analysis.Analyzer{
    Name:     "myrule",
    Doc:      "checks for...",
    Run:      runMyRule,
    Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func runMyRule(pass *analysis.Pass) (any, error) {
    // Implementation
    return nil, nil
}
```

## References

- **Source**: [microsoft/typescript-go](https://github.com/microsoft/typescript-go)
- **golangci-lint plugins**: <https://golangci-lint.run/plugins/module-plugins/>
- **Analysis framework**: <https://pkg.go.dev/golang.org/x/tools/go/analysis>
- **gotestsum**: <https://github.com/gotestyourself/gotestsum>
- **CodeQL**: <https://codeql.github.com/docs/>

## Credits

All improvements inspired by [microsoft/typescript-go](https://github.com/microsoft/typescript-go) project.

Adapted for tally with enhancements:

- CodeQL uses `security-and-quality` queries (more comprehensive)
- Custom linter focused on Dockerfile linting domain patterns
- forbidigo rules enforce buildkit usage per project philosophy
