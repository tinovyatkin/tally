package hadolint

import "fmt"

// DL3024: FROM stage names must be unique.
// This rule is checked during semantic analysis when stage names are registered.

const (
	DL3024Code   = "hadolint/DL3024"
	DL3024DocURL = "https://github.com/hadolint/hadolint/wiki/DL3024"
)

// DL3024Message formats the error message for duplicate stage names.
func DL3024Message(stageName string, existingStageIndex int) string {
	return fmt.Sprintf("Stage name %q is already used on stage %d", stageName, existingStageIndex)
}

// CheckDuplicateStageName checks if a stage name is already registered.
// Returns the existing stage index and true if duplicate found.
func CheckDuplicateStageName(stageName string, stagesByName map[string]int) (int, bool) {
	if idx, exists := stagesByName[stageName]; exists {
		return idx, true
	}
	return 0, false
}
