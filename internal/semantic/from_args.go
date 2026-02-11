package semantic

// FromArgRef represents a variable reference (e.g., $FOO) used in a FROM
// instruction that was not declared in the global ARG scope.
type FromArgRef struct {
	// Name is the referenced variable name without $ or ${}.
	Name string
	// Suggest is an optional suggested variable name.
	Suggest string
}

// FromArgsInfo contains semantic analysis results for the FROM instruction of a stage.
type FromArgsInfo struct {
	// UndefinedBaseName contains variable references used in the base image name
	// expression (the part after FROM) that are not declared in the global ARG scope.
	UndefinedBaseName []FromArgRef

	// UndefinedPlatform contains variable references used in the --platform expression
	// that are not declared in the global ARG scope.
	UndefinedPlatform []FromArgRef

	// InvalidDefaultBaseName is true when evaluating the base image expression using
	// only default values for global ARGs results in an empty or invalid image name.
	InvalidDefaultBaseName bool
}
