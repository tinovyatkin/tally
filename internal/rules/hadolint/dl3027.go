package hadolint

import (
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"

	"github.com/tinovyatkin/tally/internal/rules"
	"github.com/tinovyatkin/tally/internal/semantic"
	"github.com/tinovyatkin/tally/internal/shell"
)

// DL3027Rule implements the DL3027 linting rule.
type DL3027Rule struct{}

// NewDL3027Rule creates a new DL3027 rule instance.
func NewDL3027Rule() *DL3027Rule {
	return &DL3027Rule{}
}

// Metadata returns the rule metadata.
func (r *DL3027Rule) Metadata() rules.RuleMetadata {
	return rules.RuleMetadata{
		Code:            rules.HadolintRulePrefix + "DL3027",
		Name:            "Do not use apt",
		Description:     "Do not use apt as it is meant to be an end-user tool, use apt-get or apt-cache instead",
		DocURL:          "https://github.com/hadolint/hadolint/wiki/DL3027",
		DefaultSeverity: rules.SeverityWarning,
		Category:        "style",
		IsExperimental:  false,
	}
}

// Check runs the DL3027 rule.
// It warns when any RUN instruction contains an apt command.
// Skips analysis for stages using non-POSIX shells (e.g., PowerShell).
func (r *DL3027Rule) Check(input rules.LintInput) []rules.Violation {
	var violations []rules.Violation
	meta := r.Metadata()

	// Get semantic model for shell variant info
	sem, ok := input.Semantic.(*semantic.Model)
	if !ok {
		sem = nil
	}

	for stageIdx, stage := range input.Stages {
		// Get shell variant for this stage
		shellVariant := shell.VariantBash
		if sem != nil {
			if info := sem.StageInfo(stageIdx); info != nil {
				shellVariant = info.ShellSetting.Variant
				// Skip shell analysis for non-POSIX shells
				if shellVariant.IsNonPOSIX() {
					continue
				}
			}
		}

		for _, cmd := range stage.Commands {
			run, ok := cmd.(*instructions.RunCommand)
			if !ok {
				continue
			}

			// Check if the command contains apt using the shell package
			cmdStr := getRunCommandStringDL3027(run)
			if shell.ContainsCommandWithVariant(cmdStr, "apt", shellVariant) {
				loc := rules.NewLocationFromRanges(input.File, run.Location())
				violations = append(violations, rules.NewViolation(
					loc,
					meta.Code,
					"do not use apt as it is meant to be an end-user tool, use apt-get or apt-cache instead",
					meta.DefaultSeverity,
				).WithDocURL(meta.DocURL).WithDetail(
					"The apt command is designed for interactive use and has an unstable command-line interface. "+
						"For scripting and automation (like Dockerfiles), use apt-get for package management "+
						"or apt-cache for querying package information.",
				))
			}
		}
	}

	return violations
}

// getRunCommandStringDL3027 extracts the command string from a RUN instruction.
// Handles both shell form (RUN cmd) and exec form (RUN ["cmd", "arg"]).
func getRunCommandStringDL3027(run *instructions.RunCommand) string {
	// CmdLine contains the command parts for both shell and exec forms
	return strings.Join(run.CmdLine, " ")
}

// init registers the rule with the default registry.
func init() {
	rules.Register(NewDL3027Rule())
}
