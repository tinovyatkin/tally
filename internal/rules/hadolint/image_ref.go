package hadolint

import (
	"github.com/distribution/reference"
)

// imageRef wraps a parsed image reference providing type-safe accessors.
// It encapsulates the reference.Named and provides methods to extract
// tag, digest, domain, and path without string manipulation.
type imageRef struct {
	named reference.Named
}

// parseImageRef parses an image reference string into a structured form.
// Returns nil if the image cannot be parsed (invalid format).
// Uses ParseNormalizedNamed to handle Docker Hub shorthand (e.g., "ubuntu" -> "docker.io/library/ubuntu").
func parseImageRef(image string) *imageRef {
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return nil
	}
	return &imageRef{named: named}
}

// Tag returns the tag if present, or empty string if no tag.
// Uses type assertion on reference.Tagged interface.
func (r *imageRef) Tag() string {
	if tagged, ok := r.named.(reference.Tagged); ok {
		return tagged.Tag()
	}
	return ""
}

// HasTag returns true if the reference has an explicit tag.
func (r *imageRef) HasTag() bool {
	_, ok := r.named.(reference.Tagged)
	return ok
}

// HasDigest returns true if the reference has a digest.
func (r *imageRef) HasDigest() bool {
	_, ok := r.named.(reference.Digested)
	return ok
}

// HasExplicitVersion returns true if the reference has either a tag or digest.
// Images without explicit versions default to :latest which is unpinned.
func (r *imageRef) HasExplicitVersion() bool {
	return r.HasTag() || r.HasDigest()
}

// IsLatestTag returns true if the reference explicitly uses the :latest tag.
func (r *imageRef) IsLatestTag() bool {
	return r.Tag() == "latest"
}

// Domain returns the registry domain (e.g., "docker.io", "gcr.io").
func (r *imageRef) Domain() string {
	return reference.Domain(r.named)
}

// FamiliarName returns a familiar/shortened name for display.
// For Docker Hub images, returns just the repo name (e.g., "ubuntu" instead of "docker.io/library/ubuntu").
func (r *imageRef) FamiliarName() string {
	return reference.FamiliarName(r.named)
}
