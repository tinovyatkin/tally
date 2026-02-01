package fixes

import (
	"fmt"
	"strings"

	"github.com/tinovyatkin/tally/internal/rules"
)

// enrichFromAsCasingFix adds auto-fix for BuildKit's FromAsCasing rule.
// This fixes mismatched casing between FROM and AS keywords.
//
// Example:
//
//	FROM alpine as builder  -> FROM alpine AS builder
//	from alpine AS builder  -> from alpine as builder
func enrichFromAsCasingFix(v *rules.Violation, source []byte) {
	loc := v.Location

	// Get the line (Location uses 1-based line numbers)
	lineIdx := loc.Start.Line - 1
	if lineIdx < 0 {
		return
	}

	line := getLine(source, lineIdx)
	if line == nil {
		return
	}

	lineStr := string(line)

	// Determine FROM casing by checking if it starts with uppercase F
	fromIsUpper := strings.HasPrefix(lineStr, "FROM") || strings.HasPrefix(strings.TrimSpace(lineStr), "FROM")

	// Find AS keyword position
	asStart, asEnd, _, _ := findASKeyword(line)
	if asStart < 0 {
		return
	}

	currentAS := string(line[asStart:asEnd])
	var newAS string
	if fromIsUpper {
		newAS = "AS"
	} else {
		newAS = "as"
	}

	// Skip if already correct
	if currentAS == newAS {
		return
	}

	v.SuggestedFix = &rules.SuggestedFix{
		Description: fmt.Sprintf("Change '%s' to '%s' to match FROM casing", currentAS, newAS),
		Safety:      rules.FixSafe,
		Edits: []rules.TextEdit{{
			Location: createEditLocation(loc.File, loc.Start.Line, asStart, asEnd),
			NewText:  newAS,
		}},
	}
}
