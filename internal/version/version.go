package version

import (
	"runtime"
	"runtime/debug"
)

var version = "dev"

// Version returns the current version string with BuildKit suffix.
func Version() string {
	bkVersion := BuildKitVersion()
	if bkVersion != "" {
		return version + " (buildkit " + bkVersion + ")"
	}
	return version
}

// RawVersion returns the semantic version string without any suffix.
func RawVersion() string {
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

// GoVersion returns the Go toolchain version used for the build.
func GoVersion() string {
	return runtime.Version()
}

// GitCommit returns the VCS revision embedded at build time, if available.
func GitCommit() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			if len(s.Value) > 12 {
				return s.Value[:12]
			}
			return s.Value
		}
	}
	return ""
}

// Info holds structured version information for machine-readable output.
type Info struct {
	Version         string   `json:"version"`
	BuildkitVersion string   `json:"buildkitVersion,omitempty"`
	Platform        Platform `json:"platform"`
	GoVersion       string   `json:"goVersion"`
	GitCommit       string   `json:"gitCommit,omitempty"`
}

// Platform describes the OS and architecture.
type Platform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// GetInfo returns structured version information.
func GetInfo() Info {
	return Info{
		Version:         RawVersion(),
		BuildkitVersion: BuildKitVersion(),
		Platform: Platform{
			OS:   runtime.GOOS,
			Arch: runtime.GOARCH,
		},
		GoVersion: GoVersion(),
		GitCommit: GitCommit(),
	}
}
