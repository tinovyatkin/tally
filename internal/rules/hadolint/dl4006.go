package hadolint

import (
	"path"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"

	"github.com/tinovyatkin/tally/internal/dockerfile"
	"github.com/tinovyatkin/tally/internal/rules"
	"github.com/tinovyatkin/tally/internal/semantic"
	"github.com/tinovyatkin/tally/internal/shell"
)

// DL4006Rule implements the DL4006 linting rule.
type DL4006Rule struct{}

// NewDL4006Rule creates a new DL4006 rule instance.
func NewDL4006Rule() *DL4006Rule {
	return &DL4006Rule{}
}

// Metadata returns the rule metadata.
func (r *DL4006Rule) Metadata() rules.RuleMetadata {
	return rules.RuleMetadata{
		Code:            rules.HadolintRulePrefix + "DL4006",
		Name:            "Set pipefail",
		Description:     "Set the SHELL option -o pipefail before RUN with a pipe in it",
		DocURL:          "https://github.com/hadolint/hadolint/wiki/DL4006",
		DefaultSeverity: rules.SeverityWarning,
		Category:        "reliability",
		IsExperimental:  false,
	}
}

// pipefailValidShells are shells that support the -o pipefail option.
// Plain /bin/sh does NOT reliably support pipefail (varies by distro),
// so it is excluded. Matches hadolint's valid shells.
var pipefailValidShells = map[string]bool{
	"bash": true,
	"zsh":  true,
	"ash":  true,
}

// dl4006StageState tracks the pipefail state within a single stage.
type dl4006StageState struct {
	pipefailSet bool
	isNonPOSIX  bool
}

// Check runs the DL4006 rule.
// It warns when a RUN instruction contains a pipe (|) but the stage has not
// set -o pipefail via a SHELL instruction with a shell that supports it.
//
// State is tracked per-stage and resets on each FROM:
//   - SHELL with a non-POSIX shell (pwsh, cmd) → skip all subsequent RUNs
//   - SHELL with a valid shell and -o pipefail → mark pipefail as set
//   - SHELL with a valid shell but no pipefail → mark pipefail as not set
//   - RUN with pipes and no pipefail → violation
func (r *DL4006Rule) Check(input rules.LintInput) []rules.Violation {
	var violations []rules.Violation
	meta := r.Metadata()

	sem, ok := input.Semantic.(*semantic.Model)
	if !ok {
		sem = nil
	}

	for stageIdx, stage := range input.Stages {
		state := r.initStageState(sem, stageIdx)

		for _, cmd := range stage.Commands {
			switch c := cmd.(type) {
			case *instructions.ShellCommand:
				state.updateFromShell(c.Shell)
			case *instructions.RunCommand:
				if v := r.checkRun(c, &state, sem, stageIdx, input, meta); v != nil {
					violations = append(violations, *v)
				}
			}
		}
	}

	return violations
}

// initStageState creates the initial pipefail state for a stage.
func (r *DL4006Rule) initStageState(sem *semantic.Model, stageIdx int) dl4006StageState {
	state := dl4006StageState{}
	if sem != nil {
		if info := sem.StageInfo(stageIdx); info != nil {
			if info.ShellSetting.Source == semantic.ShellSourceDirective {
				state.isNonPOSIX = info.ShellSetting.Variant.IsNonPOSIX()
			}
		}
	}
	return state
}

// updateFromShell updates the pipefail tracking state from a SHELL instruction.
func (s *dl4006StageState) updateFromShell(shellCmd []string) {
	if isNonPOSIXShellCmd(shellCmd) {
		s.isNonPOSIX = true
		s.pipefailSet = false
	} else {
		s.isNonPOSIX = false
		s.pipefailSet = hasPipefailOption(shellCmd)
	}
}

// checkRun checks a single RUN command and returns a violation if applicable.
func (r *DL4006Rule) checkRun(
	run *instructions.RunCommand,
	state *dl4006StageState,
	sem *semantic.Model,
	stageIdx int,
	input rules.LintInput,
	meta rules.RuleMetadata,
) *rules.Violation {
	if state.isNonPOSIX || !run.PrependShell || state.pipefailSet {
		return nil
	}

	cmdStr := dockerfile.RunCommandString(run)
	shellVariant := shellVariantForStage(sem, stageIdx)

	if !shell.HasPipes(cmdStr, shellVariant) {
		return nil
	}

	loc := rules.NewLocationFromRanges(input.File, run.Location())
	v := rules.NewViolation(
		loc,
		meta.Code,
		"set the SHELL option -o pipefail before RUN with a pipe in it",
		meta.DefaultSeverity,
	).WithDocURL(meta.DocURL).WithDetail(
		"If you are using /bin/sh in an alpine image or if your shell is symlinked to busybox " +
			"then consider explicitly setting your SHELL to /bin/ash, or disable this check. " +
			`Use SHELL ["/bin/bash", "-o", "pipefail", "-c"] before the RUN instruction.`,
	)

	if fix := r.generateFix(input, run, stageIdx, shellVariant); fix != nil {
		v = v.WithSuggestedFix(fix)
	}

	return &v
}

// shellVariantForStage returns the shell variant for the given stage.
func shellVariantForStage(sem *semantic.Model, stageIdx int) shell.Variant {
	if sem != nil {
		if info := sem.StageInfo(stageIdx); info != nil {
			return info.ShellSetting.Variant
		}
	}
	return shell.VariantPOSIX
}

// isNonPOSIXShellCmd checks if a SHELL instruction sets a non-POSIX shell.
func isNonPOSIXShellCmd(shellCmd []string) bool {
	if len(shellCmd) == 0 {
		return false
	}
	return shell.VariantFromShell(shellCmd[0]).IsNonPOSIX()
}

// hasPipefailOption checks if a SHELL instruction array sets -o pipefail
// with a shell that supports it. Returns false for /bin/sh since it doesn't
// reliably support pipefail.
//
// Valid patterns:
//   - ["/bin/bash", "-o", "pipefail", "-c"]
//   - ["/bin/bash", "-eo", "pipefail", "-c"]
//   - ["/bin/bash", "-o", "errexit", "-o", "pipefail", "-c"]
//   - ["/bin/zsh", "-o", "pipefail", "-c"]
//   - ["/bin/ash", "-o", "pipefail", "-c"]
func hasPipefailOption(shellCmd []string) bool {
	if len(shellCmd) < 2 {
		return false
	}

	// Check the shell name is one that supports pipefail
	shellName := strings.ToLower(path.Base(shellCmd[0]))
	shellName = strings.TrimSuffix(shellName, ".exe")
	if !pipefailValidShells[shellName] {
		return false
	}

	// Look for -o pipefail pattern in the arguments
	args := shellCmd[1:]
	for i, arg := range args {
		// Check for standalone -o followed by pipefail
		if arg == "-o" && i+1 < len(args) && args[i+1] == "pipefail" {
			return true
		}

		// Check for combined flags like -eo, -xo, etc. followed by pipefail
		if len(arg) > 1 && strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") {
			flagChars := arg[1:]
			if strings.ContainsRune(flagChars, 'o') && i+1 < len(args) && args[i+1] == "pipefail" {
				return true
			}
		}
	}

	return false
}

// generateFix creates a suggested fix that adds a SHELL instruction with -o pipefail
// before the offending RUN instruction.
//
// When prefer-run-heredoc is enabled and the RUN is a heredoc candidate, skip the fix
// since heredoc conversion would need a different approach (shebang + set -o pipefail).
func (r *DL4006Rule) generateFix(
	input rules.LintInput,
	run *instructions.RunCommand,
	stageIdx int,
	shellVariant shell.Variant,
) *rules.SuggestedFix {
	if !run.PrependShell {
		return nil
	}

	// If prefer-run-heredoc is enabled and this command is a heredoc candidate,
	// skip the fix - heredoc conversion would handle this differently.
	if input.IsRuleEnabled(rules.HeredocRuleCode) {
		cmdStr := dockerfile.RunCommandString(run)
		if shell.IsHeredocCandidate(cmdStr, shellVariant, input.GetHeredocMinCommands()) {
			return nil
		}
	}

	runLoc := run.Location()
	if len(runLoc) == 0 {
		return nil
	}

	fixShell := r.determineFixShell(input, stageIdx)

	shellLine := `SHELL ["` + fixShell + `", "-o", "pipefail", "-c"]` + "\n"
	startLine := runLoc[0].Start.Line
	startCol := runLoc[0].Start.Character

	return &rules.SuggestedFix{
		Description: "Add SHELL with -o pipefail before RUN",
		Safety:      rules.FixSuggestion,
		Edits: []rules.TextEdit{{
			Location: rules.NewRangeLocation(
				input.File, startLine, startCol, startLine, startCol,
			),
			NewText: shellLine,
		}},
	}
}

// determineFixShell picks the shell path to use in the SHELL fix instruction.
func (r *DL4006Rule) determineFixShell(input rules.LintInput, stageIdx int) string {
	fixShell := "/bin/bash"
	if sem, ok := input.Semantic.(*semantic.Model); ok {
		if info := sem.StageInfo(stageIdx); info != nil && len(info.ShellSetting.Shell) > 0 {
			shellBase := strings.ToLower(path.Base(info.ShellSetting.Shell[0]))
			shellBase = strings.TrimSuffix(shellBase, ".exe")
			if pipefailValidShells[shellBase] {
				fixShell = info.ShellSetting.Shell[0]
			}
		}
	}
	return fixShell
}

// init registers the rule with the default registry.
func init() {
	rules.Register(NewDL4006Rule())
}
