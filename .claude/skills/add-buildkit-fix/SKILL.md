---
name: add-buildkit-fix
description: Add auto-fix support to an existing BuildKit linter rule
argument-hint: rule-name (e.g. StageNameCasing, FromAsCasing, ConsistentInstructionCasing)
disable-model-invocation: true
allowed-tools: Read, Write, Edit, Grep, Glob, Bash(go *), Bash(git status), Bash(make lint), WebFetch
---

# Add Auto-Fix to BuildKit Rule

You are adding auto-fix support to an existing BuildKit linter rule for the `tally` project.

## Rule to Add Fix: $ARGUMENTS

## Background: How BuildKit Fixes Work

BuildKit rules are detected by BuildKit's parser during Dockerfile parsing. Tally receives these as warnings and converts them to violations. To add
auto-fix support, we use a **fix enricher** pattern that post-processes violations to add `SuggestedFix` fields.

The architecture is:

```text
BuildKit Warning → NewViolationFromBuildKitWarning() → enrichBuildKitFix() → Violation with Fix
```

## Step 1: Understand the BuildKit Rule

1. Read BuildKit's rule definition to understand what it detects:
   - Use `WebFetch` to fetch the rule documentation from Docker docs
   - URL pattern: `https://docs.docker.com/go/dockerfile/rule/$ARGUMENTS/`

2. Read the existing BuildKit registry to understand the rule:
   - Read `internal/rules/buildkit/registry.go` to see rule metadata

3. Understand the warning message format by checking existing test snapshots:
   - Search for the rule name in `internal/integration/__snapshots__/`
   - The message format is critical for extracting fix information

4. Create a test Dockerfile that triggers the rule and run tally to see the violation:

   ```bash
   echo 'FROM alpine AS Builder' > /tmp/test.dockerfile
   go run . check --format json /tmp/test.dockerfile
   ```

## Step 2: Analyze Existing Fix Patterns

Read these files to understand the fix enricher pattern:

1. **Enricher Entry Point**: `internal/rules/buildkit/fixes/enricher.go`
   - The `EnrichBuildKitFixes()` function dispatches to rule-specific enrichers
   - You'll add a case for your rule here

2. **Position Helpers**: `internal/rules/buildkit/fixes/position.go`
   - Contains `getLine()`, `findASKeyword()`, `findCopyFromValue()`, `findFROMBaseName()`
   - Contains `createEditLocation()` for creating edit locations
   - Add new helpers here if needed for your rule

3. **Existing Fix Implementations**:
   - `internal/rules/buildkit/fixes/from_as_casing.go` - Simple single-edit fix
   - `internal/rules/buildkit/fixes/stage_name_casing.go` - Complex multi-edit fix with references

4. **Hadolint Fix Example**: `internal/rules/hadolint/dl3027.go`
   - Shows how to calculate precise edit positions within RUN commands
   - Uses `shell.FindAllCommandOccurrences()` for shell parsing

## Step 3: Determine Fix Complexity

### Simple Fixes (Single Edit)

For rules where the fix is a single text replacement on one line:

- FromAsCasing: Replace "as" with "AS" or vice versa
- ConsistentInstructionCasing: Replace instruction keyword casing

### Complex Fixes (Multiple Edits)

For rules requiring changes across multiple locations:

- StageNameCasing: Rename stage definition AND all references (COPY --from, FROM)
- These need the semantic model to find all references

### Async Fixes (External Data)

For rules requiring network I/O (not yet implemented for BuildKit rules):

- Image digest resolution
- Checksum verification

## Step 4: Implement the Fix Enricher

### Step 4a: Create the Enricher Function

Create a new file `internal/rules/buildkit/fixes/$ARGUMENTS_lower.go`:

```go
package fixes

import (
    "fmt"
    "regexp"
    "strings"

    "github.com/tinovyatkin/tally/internal/rules"
    "github.com/tinovyatkin/tally/internal/semantic"
)

// Extract information from BuildKit's warning message
// Example: "Stage name 'Builder' should be lowercase"
var ${ARGUMENTS}Regex = regexp.MustCompile(`...pattern matching message...`)

// enrich${ARGUMENTS}Fix adds auto-fix for BuildKit's $ARGUMENTS rule.
func enrich${ARGUMENTS}Fix(v *rules.Violation, sem *semantic.Model, source []byte) {
    // 1. Extract information from the violation message
    matches := ${ARGUMENTS}Regex.FindStringSubmatch(v.Message)
    if len(matches) < 2 {
        return
    }

    // 2. Get the source line
    lineIdx := v.Location.Start.Line - 1 // Convert 1-based to 0-based
    line := getLine(source, lineIdx)
    if line == nil {
        return
    }

    // 3. Find the position to edit using position helpers
    // Use existing helpers or create new ones in position.go

    // 4. Create the fix
    v.SuggestedFix = &rules.SuggestedFix{
        Description: "Description of what the fix does",
        Safety:      rules.FixSafe,
        Edits: []rules.TextEdit{{
            Location: createEditLocation(v.Location.File, v.Location.Start.Line, startCol, endCol),
            NewText:  "replacement text",
        }},
    }
}
```

### Step 4b: Register in Enricher

Add the case to `internal/rules/buildkit/fixes/enricher.go`:

```go
func EnrichBuildKitFixes(violations []rules.Violation, sem *semantic.Model, source []byte) {
    for i := range violations {
        v := &violations[i]
        // ... existing code ...

        ruleName := strings.TrimPrefix(v.RuleCode, rules.BuildKitRulePrefix)
        switch ruleName {
        case "StageNameCasing":
            enrichStageNameCasingFix(v, sem, source)
        case "FromAsCasing":
            enrichFromAsCasingFix(v, source)
        case "$ARGUMENTS":  // ADD THIS
            enrich${ARGUMENTS}Fix(v, sem, source)  // Pass sem if needed
        }
    }
}
```

### Step 4c: Add Position Helpers (if needed)

Add helpers to `internal/rules/buildkit/fixes/position.go`:

```go
// find${ARGUMENTS}Position locates the text to replace for $ARGUMENTS rule.
// Returns (start, end) byte offsets or (-1, -1) if not found.
func find${ARGUMENTS}Position(line []byte) (int, int) {
    // Implementation...
}
```

## Step 5: Critical Implementation Details

### Line Number Convention

**IMPORTANT**: The fix system uses 0-based line numbers internally, but BuildKit uses 1-based.

```go
// createEditLocation converts 1-based BuildKit line to 0-based for applyEdit
func createEditLocation(file string, lineNum, startCol, endCol int) rules.Location {
    return rules.NewRangeLocation(file, lineNum-1, startCol, lineNum-1, endCol)
}
```

Always use `createEditLocation()` which handles the conversion.

### Semantic Model Access

For fixes needing cross-instruction information:

```go
func enrichMyFix(v *rules.Violation, sem *semantic.Model, source []byte) {
    if sem == nil {
        return  // Can't fix without semantic model
    }

    // Use semantic model
    stageIdx, found := sem.StageIndexByName(stageName)
    for i := range sem.StageCount() {
        info := sem.StageInfo(i)
        // Check info.BaseImage, info.CopyFromRefs, etc.
    }
}
```

### Fix Safety Levels

Choose the appropriate safety level:

```go
rules.FixSafe       // Always correct, won't change behavior
rules.FixSuggestion // Likely correct but may need review
rules.FixUnsafe     // May change behavior significantly
```

Most casing fixes are `FixSafe`.

### Multi-Edit Fixes

For fixes that modify multiple locations:

```go
var edits []rules.TextEdit

// Add edit for stage definition
edits = append(edits, rules.TextEdit{
    Location: createEditLocation(file, line1, col1Start, col1End),
    NewText:  "replacement1",
})

// Add edits for all references
for _, ref := range references {
    edits = append(edits, rules.TextEdit{
        Location: createEditLocation(file, ref.Line, refStart, refEnd),
        NewText:  "replacement2",
    })
}

v.SuggestedFix = &rules.SuggestedFix{
    Description: "Description",
    Safety:      rules.FixSafe,
    Edits:       edits,
    IsPreferred: true,  // Mark as preferred fix
}
```

## Step 6: Write Unit Tests

Create or extend `internal/rules/buildkit/fixes/fixes_test.go`:

```go
func Test${ARGUMENTS}Fix(t *testing.T) {
    tests := []struct {
        name       string
        source     string
        wantFix    bool
        wantNewText string
        wantEdits  int
    }{
        {
            name:       "should fix case A",
            source:     "FROM alpine AS Builder",
            wantFix:    true,
            wantNewText: "builder",
            wantEdits:  1,
        },
        {
            name:      "should not fix when already correct",
            source:    "FROM alpine AS builder",
            wantFix:   false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            source := []byte(tt.source)

            // Parse if semantic model needed
            parseResult, err := dockerfile.Parse(bytes.NewReader(source), nil)
            require.NoError(t, err)
            sem := semantic.NewBuilder(parseResult, nil, "test.Dockerfile").Build()

            // Create violation matching BuildKit's format
            v := rules.Violation{
                Location: rules.NewRangeLocation("test.Dockerfile", 1, 0, 1, len(tt.source)),
                RuleCode: rules.BuildKitRulePrefix + "$ARGUMENTS",
                Message:  "Expected message from BuildKit",
            }

            // Call enricher
            enrich${ARGUMENTS}Fix(&v, sem, source)

            if tt.wantFix {
                require.NotNil(t, v.SuggestedFix)
                assert.Len(t, v.SuggestedFix.Edits, tt.wantEdits)
                assert.Equal(t, tt.wantNewText, v.SuggestedFix.Edits[0].NewText)
            } else {
                assert.Nil(t, v.SuggestedFix)
            }
        })
    }
}
```

## Step 7: Run Tests

```bash
# Run fix tests
go test ./internal/rules/buildkit/fixes/... -v

# Run all tests
go test ./...

# Run linter
make lint
```

## Step 8: Update Integration Test Snapshots

The fix will appear in JSON output, so snapshots need updating:

```bash
UPDATE_SNAPS=true go test ./internal/integration/...
```

## Step 9: Add Integration Test for Fix (Optional)

Add to `internal/integration/integration_test.go` in `TestFix`:

```go
{
    name:  "$ARGUMENTS-fix",
    input: "FROM alpine AS Builder\n",
    want:  "FROM alpine AS builder\n",
    args:  []string{"--fix"},
    wantApplied: 1,
},
```

## Step 10: Manual Verification

Test the fix end-to-end:

```bash
# Create test file
echo 'FROM alpine AS Builder' > /tmp/test.dockerfile

# Run with --fix
go run . check --fix /tmp/test.dockerfile

# Verify output
cat /tmp/test.dockerfile
```

## Common BuildKit Rules and Fix Strategies

| Rule | Fix Strategy | Complexity |
|------|--------------|------------|
| StageNameCasing | Rename stage + all references | Complex (multi-edit) |
| FromAsCasing | Match AS casing to FROM | Simple (single-edit) |
| ConsistentInstructionCasing | Normalize all instruction keywords | Complex (multi-edit) |
| LegacyKeyValueFormat | Convert `KEY value` to `KEY=value` | Simple (single-edit) |
| ExposeProtoCasing | Normalize protocol to lowercase | Simple (single-edit) |
| MaintainerDeprecated | Convert to LABEL maintainer | Simple (single-edit) |
| JSONArgsRecommended | Convert shell form to exec form | Complex (requires shell parsing) |
| WorkdirRelativePath | Prepend "/" to relative path | Simple (single-edit) |

## Position Helper Patterns

### Finding Keywords in FROM Line

```go
func findASKeyword(line []byte) (asStart, asEnd, nameStart, nameEnd int) {
    lineStr := string(line)
    lineUpper := strings.ToUpper(lineStr)

    idx := strings.Index(lineUpper, " AS ")
    if idx == -1 {
        return -1, -1, -1, -1
    }

    asStart := idx + 1  // Skip leading space
    asEnd := asStart + 2  // "AS" is 2 chars

    // Find stage name after AS
    nameStart := asEnd
    for nameStart < len(line) && unicode.IsSpace(rune(line[nameStart])) {
        nameStart++
    }
    // ... find nameEnd ...
}
```

### Finding COPY --from Value

```go
func findCopyFromValue(line []byte) (valueStart, valueEnd int) {
    lineUpper := strings.ToUpper(string(line))

    fromIdx := strings.Index(lineUpper, "--FROM=")
    if fromIdx == -1 {
        return -1, -1
    }

    const fromFlagLen = 7  // "--from="
    valueStart := fromIdx + fromFlagLen
    // ... find valueEnd ...
}
```

### Finding Instruction Keyword

```go
func findInstructionKeyword(line []byte) (start, end int) {
    // Skip leading whitespace
    start = 0
    for start < len(line) && unicode.IsSpace(rune(line[start])) {
        start++
    }

    // Find end of keyword
    end = start
    for end < len(line) && !unicode.IsSpace(rune(line[end])) {
        end++
    }

    return start, end
}
```

## Step 11: Update RULES.md Documentation

Update `RULES.md` to mark the rule as auto-fixable:

1. Find the rule in the BuildKit rules section
2. Add the `🔧` emoji to indicate auto-fix support

For rules in the "Captured from BuildKit Linter" table:

```markdown
| `buildkit/$ARGUMENTS` | Description | Warning | ✅🔧 Captured |
```

The legend in RULES.md shows:

- `🔧` = Auto-fixable with `tally check --fix`

## Checklist Before Completion

- [ ] BuildKit rule behavior understood (message format, trigger conditions)
- [ ] Fix enricher function created in `internal/rules/buildkit/fixes/`
- [ ] Position helpers added if needed
- [ ] Enricher registered in `enricher.go` switch statement
- [ ] Unit tests added to `fixes_test.go`
- [ ] Tests pass: `go test ./internal/rules/buildkit/fixes/... -v`
- [ ] All tests pass: `go test ./...`
- [ ] Linter passes: `make lint`
- [ ] Integration snapshots updated: `UPDATE_SNAPS=true go test ./internal/integration/...`
- [ ] Manual verification with `--fix` flag works correctly
- [ ] Fix handles edge cases (missing AS clause, already correct, etc.)
- [ ] Safety level appropriately set (FixSafe for casing fixes)
- [ ] RULES.md updated with 🔧 emoji to indicate auto-fix support

## Troubleshooting

### Fix Not Applied

1. Check line numbers: Are you using `createEditLocation()` which converts 1-based to 0-based?
2. Check column offsets: Are they 0-based byte offsets within the line?
3. Verify the fix is registered in `enricher.go`

### Wrong Position

1. Print the line content and calculated positions for debugging
2. Ensure you're accounting for any leading whitespace
3. Check that multi-byte characters aren't affecting byte offsets

### Semantic Model is nil

1. The semantic model is only available after full parsing
2. If the Dockerfile has parse errors, semantic model may be incomplete
3. Guard with `if sem == nil { return }`

### Test Failures After Adding Fix

1. Update snapshots: `UPDATE_SNAPS=true go test ./internal/integration/...`
2. Verify the fix JSON structure matches expected format
3. Check that edit locations are serializing correctly
