// Package wgetorcurl implements hadolint DL4001.
// This rule warns when both wget and curl are used in the same Dockerfile,
// as it's better to standardize on one tool to reduce image size and complexity.
package wgetorcurl

import (
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"

	"github.com/tinovyatkin/tally/internal/rules"
)

// Rule implements the DL4001 linting rule.
type Rule struct{}

// Metadata returns the rule metadata.
func (r *Rule) Metadata() rules.RuleMetadata {
	return rules.RuleMetadata{
		Code:            rules.HadolintRulePrefix + "DL4001",
		Name:            "Either wget or curl but not both",
		Description:     "Either use wget or curl but not both to reduce image size",
		DocURL:          "https://github.com/hadolint/hadolint/wiki/DL4001",
		DefaultSeverity: rules.SeverityWarning,
		Category:        "maintainability",
		IsExperimental:  false,
	}
}

// Check runs the DL4001 rule.
// It warns when both wget and curl are used in different RUN instructions.
func (r *Rule) Check(input rules.LintInput) []rules.Violation {
	var wgetLocs []rules.Location
	var curlLocs []rules.Location

	for _, stage := range input.Stages {
		for _, cmd := range stage.Commands {
			run, ok := cmd.(*instructions.RunCommand)
			if !ok {
				continue
			}

			cmdStr := strings.Join(run.CmdLine, " ")
			loc := rules.NewLocationFromRanges(input.File, run.Location())

			if usesWget(cmdStr) {
				wgetLocs = append(wgetLocs, loc)
			}
			if usesCurl(cmdStr) {
				curlLocs = append(curlLocs, loc)
			}
		}
	}

	// Only report if both are used
	if len(wgetLocs) == 0 || len(curlLocs) == 0 {
		return nil
	}

	violations := make([]rules.Violation, 0, len(curlLocs))

	// Report on all curl usages (arbitrary choice - could be wget instead)
	for _, loc := range curlLocs {
		violations = append(violations, rules.NewViolation(
			loc,
			r.Metadata().Code,
			"both wget and curl are used; pick one to reduce image size and complexity",
			r.Metadata().DefaultSeverity,
		).WithDocURL(r.Metadata().DocURL).WithDetail(
			"Using both wget and curl increases image size and maintenance burden. "+
				"Standardize on one tool. curl is generally preferred in containers "+
				"due to better scripting support and broader protocol support.",
		))
	}

	return violations
}

// usesWget checks if a command uses wget.
func usesWget(cmd string) bool {
	return containsCommand(cmd, "wget")
}

// usesCurl checks if a command uses curl.
func usesCurl(cmd string) bool {
	return containsCommand(cmd, "curl")
}

// containsCommand checks if a command string contains a specific command name.
func containsCommand(cmd, name string) bool {
	// Simple word boundary check
	words := strings.FieldsSeq(cmd)
	for word := range words {
		// Strip leading path components
		if idx := strings.LastIndex(word, "/"); idx >= 0 {
			word = word[idx+1:]
		}
		if word == name {
			return true
		}
	}
	return false
}

// New creates a new DL4001 rule instance.
func New() *Rule {
	return &Rule{}
}

// init registers the rule with the default registry.
func init() {
	rules.Register(New())
}
