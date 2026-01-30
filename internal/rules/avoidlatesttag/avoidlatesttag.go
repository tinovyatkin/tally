// Package avoidlatesttag implements hadolint DL3007.
// This rule warns when a base image uses the :latest tag,
// which can lead to unpredictable and non-reproducible builds.
package avoidlatesttag

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/distribution/reference"

	"github.com/tinovyatkin/tally/internal/rules"
)

// Rule implements the DL3007 linting rule.
type Rule struct{}

// Metadata returns the rule metadata.
func (r *Rule) Metadata() rules.RuleMetadata {
	return rules.RuleMetadata{
		Code:            rules.HadolintRulePrefix + "DL3007",
		Name:            "Avoid using :latest tag",
		Description:     "Using :latest is prone to errors if the image will ever update. Pin the version explicitly to a release tag.",
		DocURL:          "https://github.com/hadolint/hadolint/wiki/DL3007",
		DefaultSeverity: rules.SeverityWarning,
		Category:        "reproducibility",
		IsExperimental:  false,
	}
}

// Check runs the DL3007 rule.
// It warns when a FROM instruction uses an image with the :latest tag.
func (r *Rule) Check(input rules.LintInput) []rules.Violation {
	// Build a set of stage names for quick lookup
	stageNames := make(map[string]bool)
	for i, stage := range input.Stages {
		if stage.Name != "" {
			stageNames[strings.ToLower(stage.Name)] = true
		}
		// Numeric index is also valid
		stageNames[strconv.Itoa(i)] = true
	}

	var violations []rules.Violation

	for _, stage := range input.Stages {
		// Skip scratch - it's a special "no base" image
		if stage.BaseName == "scratch" {
			continue
		}

		// Skip stage references (FROM stagename)
		if stageNames[strings.ToLower(stage.BaseName)] {
			continue
		}

		// Check if image uses :latest tag
		if usesLatestTag(stage.BaseName) {
			loc := rules.NewLocationFromRanges(input.File, stage.Location)
			imageName := getImageName(stage.BaseName)
			violations = append(violations, rules.NewViolation(
				loc,
				r.Metadata().Code,
				fmt.Sprintf(
					"using :latest tag for image %q is prone to errors; pin a specific version instead (e.g., %s:22.04)",
					stage.BaseName,
					imageName,
				),
				r.Metadata().DefaultSeverity,
			).WithDocURL(r.Metadata().DocURL).WithDetail(
				"The :latest tag can change at any time, potentially breaking builds "+
					"or introducing unexpected behavior. Use a specific version tag for reproducibility.",
			))
		}
	}

	return violations
}

// usesLatestTag checks if an image reference uses the :latest tag.
func usesLatestTag(image string) bool {
	// Try to parse as a normalized named reference
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		// Can't parse - check for simple :latest suffix
		return strings.HasSuffix(image, ":latest")
	}

	// Check if it has a tag
	if tagged, ok := named.(reference.NamedTagged); ok {
		return tagged.Tag() == "latest"
	}
	return false
}

// getImageName extracts just the image name without the tag.
func getImageName(image string) string {
	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		// Fallback: strip the tag manually
		if idx := strings.LastIndex(image, ":"); idx != -1 {
			return image[:idx]
		}
		return image
	}
	return reference.FamiliarName(named)
}

// New creates a new DL3007 rule instance.
func New() *Rule {
	return &Rule{}
}

// init registers the rule with the default registry.
func init() {
	rules.Register(New())
}
