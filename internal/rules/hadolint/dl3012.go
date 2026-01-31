package hadolint

// DL3012: Multiple HEALTHCHECK instructions found.
// Only one HEALTHCHECK instruction should exist per stage.

const (
	DL3012Code    = "hadolint/DL3012"
	DL3012Message = "Multiple HEALTHCHECK instructions found in stage"
	DL3012DocURL  = "https://github.com/hadolint/hadolint/wiki/DL3012"
)
