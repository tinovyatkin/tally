package hadolint

import "strings"

// DL3061: Invalid instruction order - Dockerfile must begin with FROM, ARG, or comment.
// This rule ensures proper Dockerfile structure.

const (
	DL3061Code    = "hadolint/DL3061"
	DL3061Message = "Dockerfile must begin with FROM or ARG"
	DL3061DocURL  = "https://github.com/hadolint/hadolint/wiki/DL3061"
)

// IsValidFirstInstruction checks if an instruction is allowed as the first instruction.
// Only FROM and ARG are valid first instructions.
func IsValidFirstInstruction(instruction string) bool {
	normalized := strings.ToLower(instruction)
	return normalized == "from" || normalized == "arg"
}
