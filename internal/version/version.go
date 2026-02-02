package version

import (
	"runtime/debug"
)

var version = "dev"

// Version returns the current version string
func Version() string {
	bkVersion := BuildKitVersion()
	if bkVersion != "" {
		return version + " (buildkit " + bkVersion + ")"
	}
	return version
}

// BuildKitVersion returns the linked BuildKit version from build info.
func BuildKitVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, dep := range info.Deps {
		if dep.Path == "github.com/moby/buildkit" {
			return dep.Version
		}
	}
	return ""
}
