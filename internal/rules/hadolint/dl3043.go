package hadolint

import "strings"

// DL3043: ONBUILD, FROM, or MAINTAINER triggered from within ONBUILD instruction.
// These instructions are not allowed as ONBUILD triggers.

const (
	DL3043Code    = "hadolint/DL3043"
	DL3043Message = "`ONBUILD`, `FROM` or `MAINTAINER` triggered from within `ONBUILD` instruction."
	DL3043DocURL  = "https://github.com/hadolint/hadolint/wiki/DL3043"
)

var forbiddenOnbuildTriggers = map[string]bool{
	"onbuild":    true,
	"from":       true,
	"maintainer": true,
}

// IsForbiddenOnbuildTrigger checks if an instruction is forbidden as an ONBUILD trigger.
func IsForbiddenOnbuildTrigger(triggerInstruction string) bool {
	return forbiddenOnbuildTriggers[strings.ToLower(triggerInstruction)]
}
