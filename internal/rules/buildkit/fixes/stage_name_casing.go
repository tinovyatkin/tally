package fixes

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"

	"github.com/tinovyatkin/tally/internal/rules"
	"github.com/tinovyatkin/tally/internal/semantic"
)

// stageCasingRegex extracts the stage name from BuildKit's warning message.
// Message format: "Stage name 'Builder' should be lowercase"
var stageCasingRegex = regexp.MustCompile(`Stage name '([^']+)' should be lowercase`)

// enrichStageNameCasingFix adds auto-fix for BuildKit's StageNameCasing rule.
// This fixes stage names that should be lowercase, updating both the definition
// and all references (FROM and COPY --from).
//
// Example:
//
//	FROM alpine AS Builder    -> FROM alpine AS builder
//	COPY --from=Builder ...   -> COPY --from=builder ...
//	FROM Builder              -> FROM builder
//
func enrichStageNameCasingFix(v *rules.Violation, sem *semantic.Model, source []byte) {
	// Extract stage name from message
	matches := stageCasingRegex.FindStringSubmatch(v.Message)
	if len(matches) < 2 || sem == nil {
		return
	}

	stageName := matches[1]
	lowerName := strings.ToLower(stageName)

	// Find the stage by name
	stageIdx, found := sem.StageIndexByName(stageName)
	if !found {
		return
	}

	file := v.Location.File
	var edits []rules.TextEdit

	// 1. Fix the stage definition (FROM ... AS stagename)
	if edit := createStageDefEdit(sem.Stage(stageIdx), stageName, lowerName, file, source); edit != nil {
		edits = append(edits, *edit)
	}

	// 2. Fix all references to this stage
	edits = append(edits, collectStageRefEdits(sem, stageIdx, stageName, lowerName, file, source)...)

	if len(edits) > 0 {
		v.SuggestedFix = &rules.SuggestedFix{
			Description: fmt.Sprintf("Rename stage '%s' to '%s'", stageName, lowerName),
			Safety:      rules.FixSafe,
			Edits:       edits,
			IsPreferred: true,
		}
	}
}

// createStageDefEdit creates an edit for the stage definition (FROM ... AS stagename).
func createStageDefEdit(stage *instructions.Stage, stageName, lowerName, file string, source []byte) *rules.TextEdit {
	if stage == nil || len(stage.Location) == 0 {
		return nil
	}

	lineIdx := stage.Location[0].Start.Line - 1
	line := getLine(source, lineIdx)
	if line == nil {
		return nil
	}

	_, _, nameStart, nameEnd := findASKeyword(line)
	if nameStart < 0 || nameEnd <= nameStart {
		return nil
	}

	// Verify the name matches what we expect
	foundName := string(line[nameStart:nameEnd])
	if !strings.EqualFold(foundName, stageName) {
		return nil
	}

	return &rules.TextEdit{
		Location: createEditLocation(file, stage.Location[0].Start.Line, nameStart, nameEnd),
		NewText:  lowerName,
	}
}

// collectStageRefEdits collects edits for all references to a stage.
func collectStageRefEdits(sem *semantic.Model, stageIdx int, stageName, lowerName, file string, source []byte) []rules.TextEdit {
	var edits []rules.TextEdit

	for i := range sem.StageCount() {
		info := sem.StageInfo(i)
		if info == nil {
			continue
		}

		// Check FROM <stagename> references (multi-stage builds)
		if edit := createFromRefEdit(info, stageIdx, stageName, lowerName, file, source); edit != nil {
			edits = append(edits, *edit)
		}

		// Check COPY --from=<stagename> references
		edits = append(edits, createCopyFromEdits(info, stageIdx, stageName, lowerName, file, source)...)
	}

	return edits
}

// createFromRefEdit creates an edit for FROM <stagename> references.
func createFromRefEdit(info *semantic.StageInfo, stageIdx int, stageName, lowerName, file string, source []byte) *rules.TextEdit {
	if info.BaseImage == nil || !info.BaseImage.IsStageRef || info.BaseImage.StageIndex != stageIdx {
		return nil
	}
	if len(info.BaseImage.Location) == 0 {
		return nil
	}

	lineIdx := info.BaseImage.Location[0].Start.Line - 1
	line := getLine(source, lineIdx)
	if line == nil {
		return nil
	}

	start, end := findFROMBaseName(line)
	if start < 0 || end <= start {
		return nil
	}

	foundName := string(line[start:end])
	if !strings.EqualFold(foundName, stageName) {
		return nil
	}

	return &rules.TextEdit{
		Location: createEditLocation(file, info.BaseImage.Location[0].Start.Line, start, end),
		NewText:  lowerName,
	}
}

// createCopyFromEdits creates edits for COPY --from=<stagename> references.
func createCopyFromEdits(info *semantic.StageInfo, stageIdx int, stageName, lowerName, file string, source []byte) []rules.TextEdit {
	edits := make([]rules.TextEdit, 0, len(info.CopyFromRefs))

	for _, ref := range info.CopyFromRefs {
		if !ref.IsStageRef || ref.StageIndex != stageIdx || len(ref.Location) == 0 {
			continue
		}

		lineIdx := ref.Location[0].Start.Line - 1
		line := getLine(source, lineIdx)
		if line == nil {
			continue
		}

		valueStart, valueEnd := findCopyFromValue(line)
		if valueStart < 0 || valueEnd <= valueStart {
			continue
		}

		foundName := string(line[valueStart:valueEnd])
		if !strings.EqualFold(foundName, stageName) {
			continue
		}

		edits = append(edits, rules.TextEdit{
			Location: createEditLocation(file, ref.Location[0].Start.Line, valueStart, valueEnd),
			NewText:  lowerName,
		})
	}

	return edits
}
