package tally

import (
	"path"
	"slices"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"

	"github.com/tinovyatkin/tally/internal/rules"
	"github.com/tinovyatkin/tally/internal/rules/configutil"
	"github.com/tinovyatkin/tally/internal/semantic"
	"github.com/tinovyatkin/tally/internal/shell"
)

// PreferAddUnpackConfig is the configuration for the prefer-add-unpack rule.
type PreferAddUnpackConfig struct {
	// Enabled controls whether the rule is active. True by default.
	Enabled *bool `json:"enabled,omitempty" koanf:"enabled"`
}

// DefaultPreferAddUnpackConfig returns the default configuration.
func DefaultPreferAddUnpackConfig() PreferAddUnpackConfig {
	t := true
	return PreferAddUnpackConfig{Enabled: &t}
}

// PreferAddUnpackRule flags RUN commands that download and extract remote
// archives (via curl/wget piped to tar, or downloaded then extracted),
// suggesting `ADD --unpack <url> <dest>` instead.
//
// ADD --unpack is a BuildKit feature that downloads and extracts a remote
// archive in a single layer, reducing image size and build complexity.
type PreferAddUnpackRule struct{}

// NewPreferAddUnpackRule creates a new rule instance.
func NewPreferAddUnpackRule() *PreferAddUnpackRule {
	return &PreferAddUnpackRule{}
}

// Metadata returns the rule metadata.
func (r *PreferAddUnpackRule) Metadata() rules.RuleMetadata {
	return rules.RuleMetadata{
		Code:            rules.TallyRulePrefix + "prefer-add-unpack",
		Name:            "Prefer ADD --unpack for remote archives",
		Description:     "Use `ADD --unpack` instead of downloading and extracting remote archives in `RUN`",
		DocURL:          "https://github.com/tinovyatkin/tally/blob/main/docs/rules/tally/prefer-add-unpack.md",
		DefaultSeverity: rules.SeverityInfo,
		Category:        "performance",
		IsExperimental:  false,
		FixPriority:     95,
	}
}

// Schema returns the JSON Schema for this rule's configuration.
func (r *PreferAddUnpackRule) Schema() map[string]any {
	return map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": map[string]any{
			"enabled": map[string]any{
				"type":        "boolean",
				"default":     true,
				"description": "Enable or disable the rule",
			},
		},
		"additionalProperties": false,
	}
}

// DefaultConfig returns the default configuration.
func (r *PreferAddUnpackRule) DefaultConfig() any {
	return DefaultPreferAddUnpackConfig()
}

// ValidateConfig validates the configuration against the rule's JSON Schema.
func (r *PreferAddUnpackRule) ValidateConfig(config any) error {
	return configutil.ValidateWithSchema(config, r.Schema())
}

// Check runs the prefer-add-unpack rule.
func (r *PreferAddUnpackRule) Check(input rules.LintInput) []rules.Violation {
	cfg := r.resolveConfig(input.Config)
	if cfg.Enabled != nil && !*cfg.Enabled {
		return nil
	}

	meta := r.Metadata()

	sem, ok := input.Semantic.(*semantic.Model)
	if !ok {
		sem = nil
	}

	var violations []rules.Violation

	for stageIdx, stage := range input.Stages {
		shellVariant := shell.VariantBash
		if sem != nil {
			if info := sem.StageInfo(stageIdx); info != nil {
				shellVariant = info.ShellSetting.Variant
				if shellVariant.IsNonPOSIX() {
					continue
				}
			}
		}

		// Track the effective WORKDIR as we walk through the stage.
		// Docker default is "/" when no WORKDIR is set.
		workdir := "/"

		for _, cmd := range stage.Commands {
			if wd, ok := cmd.(*instructions.WorkdirCommand); ok {
				if path.IsAbs(wd.Path) {
					workdir = path.Clean(wd.Path)
				} else {
					workdir = path.Clean(path.Join(workdir, wd.Path))
				}
				continue
			}

			run, ok := cmd.(*instructions.RunCommand)
			if !ok {
				continue
			}

			cmdStr := GetRunCommandString(run)
			if hasRemoteArchiveExtraction(cmdStr, shellVariant) {
				loc := rules.NewLocationFromRanges(input.File, run.Location())
				v := rules.NewViolation(
					loc, meta.Code,
					"use `ADD --unpack <url> <dest>` instead of downloading and extracting in `RUN`",
					meta.DefaultSeverity,
				).WithDetail(
					"Instead of using curl/wget to download an archive and extracting it in a `RUN` command, "+
						"use `ADD --unpack <url> <dest>` which downloads and extracts in a single layer. "+
						"This reduces image size and build complexity. Requires BuildKit.",
				)

				if fix := buildAddUnpackFix(input.File, run, cmdStr, shellVariant, meta, workdir); fix != nil {
					v = v.WithSuggestedFix(fix)
				}

				violations = append(violations, v)
			}
		}
	}

	return violations
}

// resolveConfig extracts the config from input, falling back to defaults.
func (r *PreferAddUnpackRule) resolveConfig(config any) PreferAddUnpackConfig {
	return configutil.Coerce(config, DefaultPreferAddUnpackConfig())
}

// hasRemoteArchiveExtraction checks if a shell command downloads a remote archive
// and extracts it with tar, which can be replaced with ADD --unpack.
// Only tar-based extractions are detected since ADD --unpack does not handle
// single-file decompressors (gunzip, bunzip2, etc.).
func hasRemoteArchiveExtraction(cmdStr string, variant shell.Variant) bool {
	dlCmds := shell.FindCommands(cmdStr, variant, shell.DownloadCommands...)
	if len(dlCmds) == 0 {
		return false
	}

	// Check if any download command has a URL argument with an archive extension
	if !hasArchiveURLArg(dlCmds) {
		return false
	}

	// Only detect tar extraction — ADD --unpack only handles tar archives
	tarCmds := shell.FindCommands(cmdStr, variant, "tar")
	for i := range tarCmds {
		if shell.IsTarExtract(&tarCmds[i]) {
			return true
		}
	}

	return false
}

// hasArchiveURLArg checks if any download command has a URL pointing to an archive.
func hasArchiveURLArg(dlCmds []shell.CommandInfo) bool {
	return slices.ContainsFunc(dlCmds, func(dl shell.CommandInfo) bool {
		return slices.ContainsFunc(dl.Args, shell.IsArchiveURL)
	})
}

// GetRunCommandString extracts the command string from a RUN instruction.
// Re-exported here for use within the tally package.
func GetRunCommandString(run *instructions.RunCommand) string {
	return strings.Join(run.CmdLine, " ")
}

// allowedFixCommands is the set of command names that are allowed in a "simple"
// download+extract RUN instruction eligible for auto-fix. Only commands whose
// semantics are fully captured by ADD --unpack are included; other extractors
// (gunzip, unzip, etc.) would be silently dropped by the fix.
var allowedFixCommands = map[string]bool{
	"curl": true, "wget": true, // download
	"tar": true, // archive extraction (the only extractor ADD --unpack replaces)
}

// buildAddUnpackFix creates a SuggestedFix for a RUN instruction that downloads
// and extracts an archive, replacing it with ADD --unpack. Returns nil if the
// RUN contains commands beyond download+extract (not a simple case).
func buildAddUnpackFix(
	file string,
	run *instructions.RunCommand,
	cmdStr string,
	variant shell.Variant,
	meta rules.RuleMetadata,
	workdir string,
) *rules.SuggestedFix {
	url, dest, ok := extractFixData(cmdStr, variant, workdir)
	if !ok {
		return nil
	}

	runLoc := run.Location()
	if len(runLoc) == 0 {
		return nil
	}

	lastRange := runLoc[len(runLoc)-1]
	endLine := lastRange.End.Line
	endCol := lastRange.End.Character

	// Fallback: if start == end, estimate end from instruction text
	if endLine == runLoc[0].Start.Line && endCol == runLoc[0].Start.Character {
		fullInstr := "RUN " + cmdStr
		endCol = runLoc[0].Start.Character + len(fullInstr)
	}

	return &rules.SuggestedFix{
		Description: "Replace with ADD --unpack " + url + " " + dest,
		Safety:      rules.FixSuggestion,
		Priority:    meta.FixPriority,
		Edits: []rules.TextEdit{{
			Location: rules.NewRangeLocation(
				file,
				runLoc[0].Start.Line,
				runLoc[0].Start.Character,
				endLine,
				endCol,
			),
			NewText: "ADD --unpack " + url + " " + dest,
		}},
	}
}

// extractFixData checks if a RUN command is a simple download+extract and
// extracts the archive URL and destination directory.
// When tar has no explicit -C/--directory, workdir (the effective WORKDIR
// from the Dockerfile) is used as the extraction destination.
// Returns ("", "", false) if the command contains non-download/extract commands.
func extractFixData(cmdStr string, variant shell.Variant, workdir string) (string, string, bool) {
	// Check that ALL commands in the script are download or extraction commands
	allNames := shell.CommandNamesWithVariant(cmdStr, variant)
	for _, name := range allNames {
		if !allowedFixCommands[name] {
			return "", "", false
		}
	}

	// Collect all distinct archive URLs from download commands.
	// Bail out if there are multiple — we can't reliably match which
	// URL corresponds to the tar extraction (e.g. curl a.tar.gz &&
	// curl b.tar.gz && tar -xf b.tar.gz).
	var archiveURL string
	dlCmds := shell.FindCommands(cmdStr, variant, shell.DownloadCommands...)
	for _, dl := range dlCmds {
		for _, arg := range dl.Args {
			if shell.IsArchiveURL(arg) {
				if archiveURL != "" && arg != archiveURL {
					return "", "", false // multiple distinct archive URLs
				}
				archiveURL = arg
			}
		}
	}
	if archiveURL == "" {
		return "", "", false
	}

	// Only emit a fix when tar extraction is present.
	// ADD --unpack only unpacks tar archives; single-file decompressors
	// (gunzip, bunzip2, etc.) would produce an incorrect transformation.
	tarCmds := shell.FindCommands(cmdStr, variant, "tar")
	if len(tarCmds) == 0 {
		return "", "", false
	}
	// Default to the effective WORKDIR; tar without -C extracts into cwd.
	dest := workdir
	for i := range tarCmds {
		if d := shell.TarDestination(&tarCmds[i]); d != "" {
			dest = d
			break
		}
	}

	return archiveURL, dest, true
}

// init registers the rule with the default registry.
func init() {
	rules.Register(NewPreferAddUnpackRule())
}
