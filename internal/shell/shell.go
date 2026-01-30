// Package shell provides shell script parsing utilities for Dockerfile linting.
// It wraps mvdan.cc/sh/v3/syntax to provide a simple API for extracting
// command names from shell scripts, similar to how hadolint uses ShellCheck.
package shell

import (
	"path"
	"slices"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// CommandNames extracts all command names from a shell script.
// It parses the script and walks the AST to find all CallExpr nodes,
// returning the first word of each (the command name).
//
// This matches hadolint's behavior using ShellCheck.findCommandNames.
func CommandNames(script string) []string {
	// Parse the script
	parser := syntax.NewParser(
		syntax.Variant(syntax.LangBash), // Dockerfile RUN uses bash by default
		syntax.KeepComments(false),
	)

	prog, err := parser.Parse(strings.NewReader(script), "")
	if err != nil {
		// If parsing fails, fall back to simple word splitting
		return simpleCommandNames(script)
	}

	var names []string
	syntax.Walk(prog, func(node syntax.Node) bool {
		if call, ok := node.(*syntax.CallExpr); ok && len(call.Args) > 0 {
			// Get the first word (command name)
			if name := call.Args[0].Lit(); name != "" {
				// Strip path prefix (e.g., /usr/bin/wget -> wget)
				name = path.Base(name)
				names = append(names, name)
			}
		}
		return true
	})

	return names
}

// ContainsCommand checks if a shell script contains a specific command.
func ContainsCommand(script, command string) bool {
	return slices.Contains(CommandNames(script), command)
}

// simpleCommandNames is a fallback when parsing fails.
// It does basic word splitting to find potential command names.
func simpleCommandNames(script string) []string {
	var names []string

	// Replace shell operators with a marker to split on
	const marker = "\x00"
	for _, sep := range []string{"&&", "||", ";", "|", "`", "$("} {
		script = strings.ReplaceAll(script, sep, marker)
	}
	script = strings.ReplaceAll(script, "(", marker)
	script = strings.ReplaceAll(script, ")", " ")
	script = strings.ReplaceAll(script, "\\\n", " ")
	script = strings.ReplaceAll(script, "\n", marker)

	// Split by the marker to get individual command sequences
	for seq := range strings.SplitSeq(script, marker) {
		seq = strings.TrimSpace(seq)
		if seq == "" {
			continue
		}

		// Get the first non-assignment, non-flag token as the command
		for part := range strings.FieldsSeq(seq) {
			// Skip environment variable assignments (FOO=bar)
			if strings.Contains(part, "=") && !strings.HasPrefix(part, "-") {
				continue
			}
			// Skip flags
			if strings.HasPrefix(part, "-") {
				continue
			}
			// Strip path prefix
			names = append(names, path.Base(part))
			break
		}
	}

	return names
}
