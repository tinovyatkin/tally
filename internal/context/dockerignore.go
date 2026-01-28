package context

import (
	"os"
	"path/filepath"

	"github.com/moby/patternmatcher/ignorefile"
)

// dockerignoreNames are the possible names for Docker ignore files.
// .dockerignore is the standard, but we also support containerignore for Podman.
var dockerignoreNames = []string{
	".dockerignore",
	".containerignore",
}

// LoadDockerignore reads ignore patterns from the first existing ignore file
// (.dockerignore preferred, then .containerignore). Returns nil if no ignore file exists.
// An empty ignore file is valid and means "ignore no files" - we don't fall through
// to the next file in that case.
func LoadDockerignore(contextDir string) ([]string, error) {
	for _, name := range dockerignoreNames {
		ignorePath := filepath.Join(contextDir, name)
		patterns, err := loadIgnoreFile(ignorePath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		// Return patterns even if empty - an existing empty file is valid
		return patterns, nil
	}
	return nil, nil
}

// loadIgnoreFile reads patterns from a single ignore file.
func loadIgnoreFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ignorefile.ReadAll(f)
}
