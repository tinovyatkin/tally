// Package reporter provides output formatters for lint results.
//
// The text formatter is adapted from BuildKit's linter output format
// (github.com/moby/buildkit/solver/errdefs.Source.Print and
// github.com/moby/buildkit/frontend/subrequests/lint.Warning.PrintTo)
// to produce output consistent with `docker buildx build --check`.
//
// We copy the formatting logic rather than importing BuildKit's packages
// because those packages pull in heavy dependencies (containerd, grpc, etc.)
// that are unnecessary for just rendering text output.
package reporter

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/tinovyatkin/tally/internal/rules"
)

// PrintText writes violations in BuildKit's text format with source snippets.
// This produces output similar to `docker buildx build --check`.
//
// Example output:
//
//	WARNING: StageNameCasing - https://docs.docker.com/go/dockerfile/rule/stage-name-casing/
//	Stage names should be lowercase
//
//	Dockerfile:2
//	--------------------
//	   1 |     FROM ubuntu as Builder
//	   2 | >>> RUN echo hello
//	--------------------
func PrintText(w io.Writer, violations []rules.Violation, sources map[string][]byte) error {
	// Sort violations by file, then by line
	sorted := make([]rules.Violation, len(violations))
	copy(sorted, violations)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Location.File != sorted[j].Location.File {
			return sorted[i].Location.File < sorted[j].Location.File
		}
		return sorted[i].Location.Start.Line < sorted[j].Location.Start.Line
	})

	for _, v := range sorted {
		if err := printWarning(w, v, sources[v.Location.File]); err != nil {
			return err
		}
	}
	return nil
}

// printWarning formats a single warning in BuildKit style.
// Adapted from: github.com/moby/buildkit/frontend/subrequests/lint.Warning.PrintTo
func printWarning(w io.Writer, v rules.Violation, source []byte) error {
	// Header: WARNING: RuleCode - URL
	fmt.Fprintf(w, "\nWARNING: %s", v.RuleCode)
	if v.DocURL != "" {
		fmt.Fprintf(w, " - %s", v.DocURL)
	}
	fmt.Fprintf(w, "\n%s\n", v.Message)

	// Print source snippet if we have location and source
	if !v.Location.IsFileLevel() && len(source) > 0 {
		printSource(w, v.Location, source)
	}
	return nil
}

// printSource renders the source code snippet with line highlighting.
// Adapted from: github.com/moby/buildkit/solver/errdefs.Source.Print
//
// The output format matches BuildKit's style:
//   - Shows filename:line header
//   - Adds 2-4 lines of context padding
//   - Marks affected lines with ">>>" prefix
//   - Uses 1-based line numbers in display (internally 0-based)
func printSource(w io.Writer, loc rules.Location, source []byte) {
	lines := strings.Split(string(source), "\n")

	// Get start/end lines (convert from 0-based to 1-based for display)
	start := loc.Start.Line + 1 // Convert to 1-based
	end := loc.End.Line + 1
	if loc.IsPointLocation() || end < start {
		end = start
	}

	// Bounds check
	if start > len(lines) || start < 1 {
		return
	}
	if end > len(lines) {
		end = len(lines)
	}

	// Calculate padding (2-4 lines of context)
	pad := 2
	if end == start {
		pad = 4
	}

	displayStart := start // The line to show in header
	p := 0
	for p < pad {
		if start > 1 {
			start--
			p++
		}
		if end < len(lines) {
			end++
			p++
		}
		p++
	}

	// Print the snippet
	fmt.Fprintf(w, "%s:%d\n", loc.File, displayStart)
	fmt.Fprintf(w, "--------------------\n")
	for i := start; i <= end; i++ {
		pfx := "   "
		// Check if this line is in the affected range (convert back to 1-based comparison)
		if lineInRange(i, loc.Start.Line+1, loc.End.Line+1) {
			pfx = ">>>"
		}
		fmt.Fprintf(w, " %3d | %s %s\n", i, pfx, lines[i-1])
	}
	fmt.Fprintf(w, "--------------------\n")
}

// lineInRange checks if a 1-based line number is within the range [start, end].
func lineInRange(line, start, end int) bool {
	if end < start {
		end = start
	}
	return line >= start && line <= end
}
