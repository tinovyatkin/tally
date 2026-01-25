# Implementation Roadmap

Prioritized action plan based on architectural research.

This roadmap provides the next 10 critical steps to transform tally from a single-rule demo into a production-ready Dockerfile linter.

---

## Critical Insight: Dockerfiles Are Small

The research (01-linter-pipeline-architecture.md) notes that **Dockerfiles are typically < 200 lines**. This fundamentally shapes our architecture:

- ✅ Single-pass analysis is sufficient (no incremental parsing needed)
- ✅ Performance optimization (parallelism, caching) is lower priority
- ✅ Focus on correctness and rule coverage over speed
- ✅ Memory usage is not a concern for individual files

---

## Code Reuse Opportunities

Before building from scratch, consider these existing libraries:

| Library | Use For | Notes |
|---------|---------|-------|
| `moby/buildkit/frontend/dockerfile/linter` | Study rule patterns | BuildKit has 22 working rules |
| `moby/buildkit/frontend/dockerfile/dockerignore` | .dockerignore parsing | Docker-compatible pattern matching |
| `github.com/owenrumney/go-sarif/v2` | SARIF output | Already planned |
| `github.com/charmbracelet/lipgloss` | Terminal styling | Already planned |
| `github.com/bmatcuk/doublestar/v4` | Glob patterns | For file discovery |

**Note:** The BuildKit linter in `moby/buildkit/frontend/dockerfile/linter/` could potentially be imported directly for rules that overlap with their
22 built-in checks.

---

## Priority 1: Restructure Rule System + Test Utilities

**Goal:** Establish scalable rule architecture with testing foundation

**Actions:**

1. Create `internal/testutil/` package FIRST (needed for all rule testing):

   ```go
   // internal/testutil/lint.go
   func LintString(dockerfile string, rules ...*Rule) []Violation
   func LintFile(path string, rules ...*Rule) []Violation
   func ParseString(dockerfile string) (*parser.Result, error)
   ```

2. Create `internal/rules/` directory structure:

   ```text
   internal/rules/
   ├── registry.go          # Rule registration
   ├── rule.go              # Rule interface + Severity enum
   ├── violation.go         # Violation struct
   └── style/
       ├── max_lines.go     # Move existing rule here
       └── max_lines_test.go
   ```

3. Define core interfaces in `internal/rules/rule.go`:

   ```go
   type Severity int

   const (
       SeverityError Severity = iota   // Critical, must fix
       SeverityWarning                  // Important, should fix
       SeverityInfo                     // Suggestion
       SeverityStyle                    // Cosmetic
   )

   type Rule struct {
       Code        string
       Name        string
       Description string
       Category    string    // "security", "best-practices", "style", etc.
       Severity    Severity
       URL         string
       Enabled     bool      // Default enabled state
       Check       RuleFunc
   }

   type RuleFunc func(ast *parser.Result, semantic *SemanticModel) []Violation
   ```

4. Implement auto-registration pattern (init() functions)

5. Move `max-lines` rule to new structure as template

**References:**

- [06-code-organization.md](06-code-organization.md) - Section "One File Per Rule"
- [06-code-organization.md](06-code-organization.md) - Section "Rule Registry"
- [09-hadolint-research.md](09-hadolint-research.md) - Section "Severity Levels"

**Success Criteria:**

- [ ] `internal/testutil/` package created with LintString helper
- [ ] Rule interface defined with Severity enum
- [ ] Registry implemented with Register() and GetRule() functions
- [ ] max-lines rule migrated to new structure
- [ ] Tests pass using new testutil helpers

---

## Priority 2: Build Semantic Model

**Goal:** Enable advanced rules that need cross-instruction context

**Key Insight:** Some violations (like DL3024 - duplicate stage names) should be detected DURING semantic model construction, not as separate rules.
This is more efficient and architecturally cleaner.

**Actions:**

1. Create `internal/parser/semantic.go`:

   ```go
   type SemanticModel struct {
       // Stage management
       Stages          map[string]*Stage    // Stage name → first definition
       StageOrder      []string             // Preserves declaration order
       DuplicateStages []DuplicateStage     // Track duplicates for DL3024

       // Variable scoping
       GlobalArgs  map[string]*Variable           // ARG before first FROM
       StageVars   map[string]map[string]*Variable // Per-stage ARG/ENV
       BuildArgs   map[string]string              // CLI --build-arg overrides

       // Cross-stage references
       CopyFromRefs []CopyFromRef  // COPY --from references

       // Base images
       BaseImages []BaseImageRef

       // Construction errors (violations detected during building)
       ConstructionViolations []Violation
   }

   type DuplicateStage struct {
       Name           string
       FirstLine      int
       DuplicateLine  int
   }
   ```

2. Implement `BuildSemanticModel(ast, buildArgs)` function:
   - Parse FROM instructions → stages (detect duplicates immediately)
   - Track ARG/ENV → variables with scopes
   - Support BuildArgs from CLI for proper variable resolution
   - Collect COPY --from → cross-stage references
   - Store base images with platforms
   - Return construction violations (DL3024, etc.)

3. Add `ResolveVariable(name, stage)` method with proper precedence:
   1. BuildArgs (CLI --build-arg, highest priority)
   2. Stage-local variables (ARG/ENV in current stage)
   3. Global ARGs (ARG before first FROM)

4. Update linter to pass semantic model to rules

**References:**

- [03-parsing-and-ast.md](03-parsing-and-ast.md) - Section "Semantic Analysis"
- [03-parsing-and-ast.md](03-parsing-and-ast.md) - Section "Building the Semantic Model"
- [01-linter-pipeline-architecture.md](01-linter-pipeline-architecture.md) - Section "2. Parsing Stage"

**Success Criteria:**

- [ ] SemanticModel struct defined with all fields
- [ ] BuildSemanticModel() implemented with duplicate detection
- [ ] ResolveVariable() method with correct precedence
- [ ] DL3024 (duplicate stage names) detected during construction
- [ ] Unit tests for semantic analysis
- [ ] Can track stages, variables, and references

---

## Priority 3: Implement Inline Disable Support

**Goal:** Allow users to suppress specific violations with migration compatibility

**Key Insight:** Support multiple syntax formats for easy migration from hadolint and buildx.

**Actions:**

1. Create `internal/inline/` package:
   - `directive.go` - Parse inline comments
   - `filter.go` - Filter violations based on directives

2. Support multiple syntax formats (migration compatibility):

   ```dockerfile
   # Primary tally syntax
   # tally ignore=DL3006
   # tally ignore=DL3006,DL3008
   # tally global ignore=DL3003
   # tally ignore=all

   # Hadolint compatibility (for migration)
   # hadolint ignore=DL3006,DL3008
   # hadolint global ignore=DL3003

   # BuildKit/buildx compatibility
   # check=skip=DL3006,DL3008
   ```

3. Implement directive parsing with regex patterns:

   ```go
   var patterns = []struct {
       pattern *regexp.Regexp
       parser  func(matches []string) *Directive
   }{
       {tallyIgnorePattern, parseTallyIgnore},
       {hadolintIgnorePattern, parseHadolintIgnore},  // Migration
       {checkSkipPattern, parseCheckSkip},            // Buildx compat
   }
   ```

4. Add post-filtering step to linter pipeline

5. Track unused directives for warnings (optional rule)

6. Validate rule codes in directives (warn on unknown codes)

**References:**

- [04-inline-disables.md](04-inline-disables.md) - Section "Recommended Implementation for Tally"
- [04-inline-disables.md](04-inline-disables.md) - Section "Full Implementation"
- [09-hadolint-research.md](09-hadolint-research.md) - Section "Inline Disable Mechanism"

**Success Criteria:**

- [ ] Can parse `# tally ignore=` syntax
- [ ] Can parse `# hadolint ignore=` syntax (migration)
- [ ] Can parse `# check=skip=` syntax (buildx compat)
- [ ] Filter() removes suppressed violations
- [ ] Detect unused directives
- [ ] Warn on unknown rule codes
- [ ] Integration tests with all syntax variants

---

## Priority 4: Create Reporter Infrastructure + CI Formats

**Goal:** Support multiple output formats including CI/CD integration

**Actions:**

1. Create `internal/reporter/` package with interface:

   ```go
   type Reporter interface {
       Report(violations []Violation, summary Summary) error
   }

   type Summary struct {
       Total    int
       Errors   int
       Warnings int
       Files    int
   }
   ```

2. Implement reporters (most to least important):
   - `text.go` - Human-readable colored output (use Lip Gloss)
   - `json.go` - Machine-readable structured output
   - `github_actions.go` - Native GitHub annotations (`::error file=...`)
   - `sarif.go` - SARIF 2.1.0 format (use go-sarif)

3. Add factory pattern for format selection

4. Wire into CLI with `--format` flag

5. **Define exit codes:**
   - `0` - No violations (or only info/style with default threshold)
   - `1` - Violations found at or above threshold
   - `2` - Configuration/parse error

6. Add `--fail-level` flag to control exit code threshold:
   - `--fail-level=error` (default) - Exit 1 only on errors
   - `--fail-level=warning` - Exit 1 on warnings or errors
   - `--fail-level=none` - Always exit 0 (for CI that handles output)

**References:**

- [05-reporters-and-output.md](05-reporters-and-output.md) - Section "Core Reporter Pattern"
- [05-reporters-and-output.md](05-reporters-and-output.md) - Section "Multiple Output Support"
- [02-buildx-bake-check-analysis.md](02-buildx-bake-check-analysis.md) - Section "Exit Codes"

**Success Criteria:**

- [ ] Reporter interface defined
- [ ] Text reporter with colors (Lip Gloss)
- [ ] JSON reporter with summary
- [ ] GitHub Actions reporter (`::error` format)
- [ ] SARIF reporter (replaces Priority 9)
- [ ] Factory for format selection
- [ ] CLI flag `--format=text|json|github-actions|sarif`
- [ ] Exit codes documented and tested
- [ ] `--fail-level` flag implemented

---

## Priority 5: Implement File Discovery

**Goal:** Find all Dockerfiles to lint (prerequisite for multi-file linting)

**Rationale:** File discovery must come BEFORE parallelism - you need files to parallelize over. Since Dockerfiles are small (< 200 lines),
single-file performance is already fast; the bottleneck is finding and processing many files.

**Actions:**

1. Create `internal/discovery/` package

2. Support input types:
   - Single file: `tally check Dockerfile`
   - Directory: `tally check .` (find all Dockerfiles recursively)
   - Multiple: `tally check Dockerfile build/Dockerfile.prod`
   - Glob patterns: `tally check **/Dockerfile*`

3. Use `github.com/bmatcuk/doublestar/v4` for glob pattern matching

4. Filter logic:
   - Skip hidden directories (unless explicit)
   - Respect `.gitignore` (optional `--respect-gitignore` flag)
   - Default Dockerfile patterns: `Dockerfile`, `Dockerfile.*`, `*.Dockerfile`

5. Add `--exclude` flag for patterns

6. Use BuildKit's dockerignore package for pattern matching:

   ```go
   import "github.com/moby/buildkit/frontend/dockerfile/dockerignore"
   ```

**References:**

- [01-linter-pipeline-architecture.md](01-linter-pipeline-architecture.md) - Section "1. File Discovery Stage"
- [07-context-aware-foundation.md](07-context-aware-foundation.md) - Section ".dockerignore parsing"

**Success Criteria:**

- [ ] Can discover files from various inputs
- [ ] Recursive directory search works
- [ ] Glob patterns supported (doublestar)
- [ ] Exclusion patterns work
- [ ] Uses BuildKit's dockerignore for pattern matching

---

## Priority 6: Implement Top 5 Critical Rules

**Goal:** Provide immediate value with essential rules

**Note:** DL3024 (duplicate stage names) is now detected during semantic model construction (Priority 2), not as a separate rule.

**Actions:**
Implement these rules (one file each in appropriate category):

1. **DL3006** - Pin base image versions (`internal/rules/base/pin_version.go`)
   - Check FROM instructions lack explicit tag
   - Severity: warning

2. **DL3004** - No sudo (`internal/rules/security/no_sudo.go`)
   - Scan RUN instructions for sudo usage
   - Severity: error

3. **DL3020** - Use COPY not ADD (`internal/rules/instruction/copy_not_add.go`)
   - Check for ADD when COPY is appropriate
   - Severity: error

4. **DL3002** - Don't run as root (`internal/rules/security/no_root_user.go`)
   - Check last USER instruction is not root
   - Severity: warning

5. **DL4000** - MAINTAINER deprecated (`internal/rules/deprecation/maintainer.go`)
   - Flag any MAINTAINER instruction (use LABEL instead)
   - Severity: error

Each rule needs:

- Implementation file following existing patterns
- Test file with table-driven tests
- Examples (good/bad Dockerfiles in tests)

**Rule Implementation Pattern** (from research):

```go
// internal/rules/security/no_sudo.go
package security

func init() {
    rules.Register(NoSudoRule)
}

var NoSudoRule = &rules.Rule{
    Code:        "DL3004",
    Name:        "Do not use sudo",
    Description: "Using sudo has unpredictable behavior in a Dockerfile",
    Category:    "security",
    Severity:    rules.SeverityError,
    URL:         "https://github.com/hadolint/hadolint/wiki/DL3004",
    Enabled:     true,
    Check:       checkNoSudo,
}

func checkNoSudo(ast *parser.Result, semantic *parser.SemanticModel) []rules.Violation {
    // Implementation
}
```

**References:**

- [08-hadolint-rules-reference.md](08-hadolint-rules-reference.md) - Section "Critical Priority"
- [06-code-organization.md](06-code-organization.md) - Section "Rule Structure Template"
- [09-hadolint-research.md](09-hadolint-research.md) - Section "Rule Implementation Pattern"

**Success Criteria:**

- [ ] All 5 rules implemented
- [ ] Unit tests for each rule (using testutil)
- [ ] Integration tests pass
- [ ] Rules auto-register via init()

---

## Priority 7: Add Violation Processing Pipeline + Severity Configuration

**Goal:** Filter, deduplicate, sort violations and support severity overrides

**Actions:**

1. Create processor chain in `internal/linter/pipeline.go`:

   ```go
   type Processor interface {
       Process(violations []Violation) ([]Violation, error)
   }
   ```

2. Implement processors:
   - `InlineDisableFilter` - Apply `# tally ignore=` directives
   - `SeverityOverrider` - Apply config-based severity changes
   - `Deduplicator` - Remove exact duplicates
   - `SortProcessor` - Sort by severity, file, line
   - `MaxPerFileFilter` - Limit violations per file (configurable)

3. Chain processors in linter

4. Add severity override configuration support:

   ```toml
   # .tally.toml
   [rules.DL3006]
   severity = "error"    # Upgrade from warning

   [rules.DL3002]
   enabled = false       # Disable this rule
   ```

5. Update config loading to support per-rule configuration

**References:**

- [01-linter-pipeline-architecture.md](01-linter-pipeline-architecture.md) - Section "5. Processing Pipeline"
- [04-inline-disables.md](04-inline-disables.md) - Section "Approach 3: Post-Filtering"
- [09-hadolint-research.md](09-hadolint-research.md) - Section "Configuration System"

**Success Criteria:**

- [ ] Processor interface defined
- [ ] Core processors implemented
- [ ] Pipeline chains processors
- [ ] Violations are filtered and sorted
- [ ] Severity override configuration works
- [ ] Per-rule enable/disable configuration works

---

## Priority 8: Add Rule CLI Commands

**Goal:** Enable rule discovery and management

**Actions:**

1. Add `tally rules` subcommand with subcommands:

   ```bash
   # List all rules
   tally rules list
   tally rules list --category=security
   tally rules list --enabled=false

   # Show rule details
   tally rules show DL3006

   # Output formats
   tally rules list --format=json
   ```

2. Implement in `cmd/tally/cmd/rules.go`:

   ```go
   func listRules(ctx context.Context, cmd *cli.Command) error {
       for _, rule := range rules.AllRules {
           fmt.Printf("%s: %s [%s] %s\n",
               rule.Code, rule.Name, rule.Category, rule.Severity)
       }
       return nil
   }

   func showRule(ctx context.Context, cmd *cli.Command) error {
       code := cmd.Args().Get(0)
       rule := rules.GetRule(code)
       // Print detailed info including description, URL, examples
   }
   ```

3. Add JSON output option for tooling integration

**References:**

- [06-code-organization.md](06-code-organization.md) - Section "Rule Discoverability"
- [09-hadolint-research.md](09-hadolint-research.md) - Section "Rule Documentation"

**Success Criteria:**

- [ ] `tally rules list` shows all rules
- [ ] `tally rules show DL3006` shows rule details
- [ ] Filter by category, severity, enabled state
- [ ] JSON output for tooling

---

## Priority 9: Add File-Level Parallelism (Optional)

**Goal:** Efficiently lint multiple files in large repositories

**Rationale:** Since Dockerfiles are small (< 200 lines), single-file performance is already fast. Parallelism only helps when linting many files
(e.g., monorepos with 100+ Dockerfiles). This is lower priority than correctness features.

**Actions:**

1. Implement worker pool in linter:

   ```go
   func (l *Linter) LintFiles(paths []string) ([]Violation, error) {
       // Use errgroup for parallel execution
       // Each file linted independently
       // Aggregate violations
   }
   ```

2. Use `golang.org/x/sync/errgroup` for coordination

3. Make worker count configurable:
   - Default: `min(len(paths), runtime.NumCPU())`
   - Flag: `--parallel=N`

4. Ensure no shared mutable state between workers

**References:**

- [01-linter-pipeline-architecture.md](01-linter-pipeline-architecture.md) - Section "Option A: File-Level Parallelism"

**Success Criteria:**

- [ ] Can lint multiple files in parallel
- [ ] Worker pool limits concurrency
- [ ] No race conditions (`go test -race` passes)
- [ ] Benchmark shows improvement for 10+ files

---

## Priority 10: Enhance Integration Tests

**Goal:** Ensure end-to-end correctness

**Actions:**

1. Expand `internal/integration/testdata/` with fixtures:

   ```text
   testdata/
   ├── critical_violations/
   │   └── Dockerfile (triggers DL3004, DL3020)
   ├── multi_stage/
   │   ├── Dockerfile (tests stage rules, DL3024)
   │   └── expected.json
   ├── with_inline_disables/
   │   ├── Dockerfile (tests # tally ignore=)
   │   └── Dockerfile.hadolint (tests # hadolint ignore=)
   ├── clean/
   │   └── Dockerfile (no violations)
   └── severity_override/
       ├── Dockerfile
       └── .tally.toml (tests severity config)
   ```

2. Add snapshot tests for each fixture:
   - JSON output
   - Violation counts
   - Rule codes triggered
   - Exit codes

3. Test all reporters:
   - Text output (no color for snapshot)
   - JSON output
   - SARIF output
   - GitHub Actions output

4. Test exit codes:
   - Exit 0 on clean files
   - Exit 1 on violations
   - Exit 2 on parse errors

5. Add `make test-integration` target

**References:**

- [06-code-organization.md](06-code-organization.md) - Section "Testing Strategy" → "Integration Tests"
- Current: `internal/integration/integration_test.go`

**Success Criteria:**

- [ ] 5+ integration test fixtures
- [ ] Snapshot tests with go-snaps
- [ ] All reporters tested end-to-end
- [ ] `UPDATE_SNAPS=true make test` updates snapshots

---

## Implementation Notes

### Order Dependencies (Updated)

```text
Priority 1 (Rules + Testutil)
    ↓
Priority 2 (Semantic Model)  ←─────────────────────┐
    ↓                                              │
Priority 3 (Inline Disables)                       │
    ↓                                              │
Priority 4 (Reporters + CI formats)                │
    ↓                                              │
Priority 5 (File Discovery)                        │
    ↓                                              │
Priority 6 (Critical Rules) ───────────────────────┘
    ↓
Priority 7 (Pipeline + Severity Config)
    ↓
Priority 8 (Rule CLI)
    ↓
Priority 9 (Parallelism) ─── Optional for v1.0
    ↓
Priority 10 (Integration Tests)
```

**Key dependencies:**

- **1 → All**: Testutil and rule interface needed everywhere
- **2 → 6**: Semantic model needed for stage-aware rules
- **3 → 7**: Inline disables integrated into pipeline
- **4 → 10**: Reporters needed for integration test output
- **5 → 9**: File discovery needed before parallelism

### Key Design Principles

1. **Incremental value**: Each step produces working software
2. **Test-driven**: Add tests alongside implementation (testutil in Priority 1!)
3. **Reuse over reinvent**: Use BuildKit packages where possible
4. **Correctness over speed**: Parallelism is optional for v1.0
5. **Migration-friendly**: Support hadolint/buildx syntax for easy adoption
6. **Real-world ready**: Focus on rules users actually need

### Post-Priority 10

After completing these 10 priorities, tally will have:

- ✅ Scalable rule system with auto-registration
- ✅ Semantic analysis with duplicate stage detection
- ✅ Inline disables (tally, hadolint, buildx syntax)
- ✅ Multiple output formats (text, JSON, SARIF, GitHub Actions)
- ✅ 5 critical rules (DL3006, DL3004, DL3020, DL3002, DL4000)
- ✅ Processing pipeline with severity overrides
- ✅ File discovery with glob patterns
- ✅ Rule CLI commands
- ✅ Optional file-level parallelism
- ✅ Comprehensive integration tests

**Next phases** (reference [08-hadolint-rules-reference.md](08-hadolint-rules-reference.md)):

- Phase 2: Add 15 high-priority rules (package managers, multi-stage)
- Phase 3: Add 20 medium-priority rules (best practices)
- Phase 4: Context-aware rules ([07-context-aware-foundation.md](07-context-aware-foundation.md))
- Phase 5: ShellCheck integration for RUN instruction validation

---

## Quick Reference

| Priority | Focus | Key File(s) | Doc Reference |
|----------|-------|-------------|---------------|
| 1 | Rule system + Testutil | `internal/rules/`, `internal/testutil/` | [06](06-code-organization.md) |
| 2 | Semantic model | `internal/parser/semantic.go` | [03](03-parsing-and-ast.md) |
| 3 | Inline disables | `internal/inline/` | [04](04-inline-disables.md), [09](09-hadolint-research.md) |
| 4 | Reporters + CI | `internal/reporter/` | [05](05-reporters-and-output.md) |
| 5 | File discovery | `internal/discovery/` | [01](01-linter-pipeline-architecture.md) |
| 6 | Critical rules | `internal/rules/*/` | [08](08-hadolint-rules-reference.md) |
| 7 | Pipeline + Config | `internal/linter/pipeline.go` | [01](01-linter-pipeline-architecture.md) |
| 8 | Rule CLI | `cmd/tally/cmd/rules.go` | [06](06-code-organization.md) |
| 9 | Parallelism | `internal/linter/` | [01](01-linter-pipeline-architecture.md) |
| 10 | Integration tests | `internal/integration/` | [06](06-code-organization.md) |

---

## Tracking Progress

Create a GitHub project or use this checklist to track implementation:

```markdown
## v1.0 Implementation Checklist

### Foundation

- [ ] Priority 1: Rule system + test utilities
- [ ] Priority 2: Semantic model with DL3024 detection
- [ ] Priority 3: Inline disable support (tally/hadolint/buildx)
- [ ] Priority 4: Reporters (text, JSON, SARIF, GitHub Actions)
- [ ] Priority 5: File discovery

### Core Features

- [ ] Priority 6: Top 5 critical rules
- [ ] Priority 7: Processing pipeline + severity config
- [ ] Priority 8: Rule CLI commands
- [ ] Priority 9: File-level parallelism (optional)
- [ ] Priority 10: Integration tests

### Ready for v1.0 Release

- [ ] All priorities complete
- [ ] Exit codes documented (0/1/2)
- [ ] Documentation updated
- [ ] Examples added to README
- [ ] Release notes written
```
