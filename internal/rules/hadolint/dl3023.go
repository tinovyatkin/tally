package hadolint

import "fmt"

// DL3023: COPY --from should not reference the stage's own FROM alias.
// This rule detects when a COPY instruction tries to copy from itself.

const (
	DL3023Code   = "hadolint/DL3023"
	DL3023DocURL = "https://github.com/hadolint/hadolint/wiki/DL3023"
)

// DL3023Message formats the error message for self-referencing COPY --from.
func DL3023Message(stageName, copyFrom string) string {
	return fmt.Sprintf("COPY --from=%s references its own stage %q", copyFrom, stageName)
}

// IsSelfReferencingCopy checks if a COPY --from references the current stage.
// stageNames maps normalized stage names to their indices.
func IsSelfReferencingCopy(currentStageIndex int, copyFrom string, stageNames map[string]int) bool {
	if refIdx, exists := stageNames[copyFrom]; exists && refIdx == currentStageIndex {
		return true
	}
	return false
}
