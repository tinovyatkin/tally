package version

var (
	version = "dev"
	commit  = "unknown"
)

// Version returns the current version string
func Version() string {
	if commit != "unknown" && len(commit) > 7 {
		return version + " (" + commit[:7] + ")"
	}
	return version
}

// Commit returns the git commit hash
func Commit() string {
	return commit
}
