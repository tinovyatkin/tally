package hadolint

import (
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"

	"github.com/tinovyatkin/tally/internal/rules"
	"github.com/tinovyatkin/tally/internal/semantic"
	"github.com/tinovyatkin/tally/internal/shell"
)

// DL3004Rule implements the DL3004 linting rule.
type DL3004Rule struct{}

// NewDL3004Rule creates a new DL3004 rule instance.
func NewDL3004Rule() *DL3004Rule {
	return &DL3004Rule{}
}

// Metadata returns the rule metadata.
func (r *DL3004Rule) Metadata() rules.RuleMetadata {
	return rules.RuleMetadata{
		Code:            rules.HadolintRulePrefix + "DL3004",
		Name:            "Do not use sudo",
		Description:     "Do not use sudo as it has unpredictable behavior in containers",
		DocURL:          "https://github.com/hadolint/hadolint/wiki/DL3004",
		DefaultSeverity: rules.SeverityError,
		Category:        "security",
		IsExperimental:  false,
	}
}

// Check runs the DL3004 rule.
// It warns when any RUN instruction contains a sudo command.
// Skips analysis for stages using non-POSIX shells (e.g., PowerShell).
func (r *DL3004Rule) Check(input rules.LintInput) []rules.Violation {
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

			// Check if the command contains sudo using the shell package
			cmdStr := getRunCommandStringDL3004(run)
			if shell.ContainsCommandWithVariant(cmdStr, "sudo", shellVariant) {
				loc := rules.NewLocationFromRanges(input.File, run.Location())
				violations = append(violations, rules.NewViolation(
					loc,
					meta.Code,
					"do not use sudo in RUN commands; it has unpredictable TTY and signal handling",
					meta.DefaultSeverity,
				).WithDocURL(meta.DocURL).WithDetail(
					"sudo is designed for interactive use and doesn't work reliably in containers. "+
						"Instead, use the USER instruction to switch users, or run specific commands "+
						"as a different user with 'su -c' if necessary.",
				))
			}
		}
	}

	return violations
}

// getRunCommandStringDL3004 extracts the command string from a RUN instruction.
// Handles both shell form (RUN cmd) and exec form (RUN ["cmd", "arg"]).
func getRunCommandStringDL3004(run *instructions.RunCommand) string {
	// CmdLine contains the command parts for both shell and exec forms
	return strings.Join(run.CmdLine, " ")
}

// init registers the rule with the default registry.
func init() {
	rules.Register(NewDL3004Rule())
}
