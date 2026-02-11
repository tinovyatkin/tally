package shell

import (
	"slices"
	"testing"
)

func TestFindCommandInChain_Standalone(t *testing.T) {
	t.Parallel()
	pos := FindCommandInChain("ln -sf /bin/bash /bin/sh", VariantBash, func(name string, args []string) bool {
		return name == "ln" && slices.Contains(args, "/bin/sh")
	})
	if pos == nil {
		t.Fatal("expected to find command")
	}
	if !pos.IsStandalone {
		t.Error("expected IsStandalone to be true")
	}
	if pos.PrecedingCommands != "" {
		t.Errorf("expected empty PrecedingCommands, got %q", pos.PrecedingCommands)
	}
	if pos.RemainingCommands != "" {
		t.Errorf("expected empty RemainingCommands, got %q", pos.RemainingCommands)
	}
}

func TestFindCommandInChain_AtEnd(t *testing.T) {
	t.Parallel()
	pos := FindCommandInChain("apt-get update && ln -sf /bin/bash /bin/sh", VariantBash, func(name string, args []string) bool {
		return name == "ln" && slices.Contains(args, "/bin/sh")
	})
	if pos == nil {
		t.Fatal("expected to find command")
	}
	if pos.IsStandalone {
		t.Error("expected IsStandalone to be false")
	}
	if pos.PrecedingCommands != "apt-get update" {
		t.Errorf("PrecedingCommands = %q, want %q", pos.PrecedingCommands, "apt-get update")
	}
	if pos.RemainingCommands != "" {
		t.Errorf("expected empty RemainingCommands, got %q", pos.RemainingCommands)
	}
}

func TestFindCommandInChain_AtStart(t *testing.T) {
	t.Parallel()
	pos := FindCommandInChain("ln -sf /bin/bash /bin/sh && echo done", VariantBash, func(name string, args []string) bool {
		return name == "ln" && slices.Contains(args, "/bin/sh")
	})
	if pos == nil {
		t.Fatal("expected to find command")
	}
	if pos.PrecedingCommands != "" {
		t.Errorf("expected empty PrecedingCommands, got %q", pos.PrecedingCommands)
	}
	if pos.RemainingCommands != "echo done" {
		t.Errorf("RemainingCommands = %q, want %q", pos.RemainingCommands, "echo done")
	}
}

func TestFindCommandInChain_InMiddle(t *testing.T) {
	t.Parallel()
	pos := FindCommandInChain(
		"apt-get update && ln -sf /bin/bash /bin/sh && echo done",
		VariantBash,
		func(name string, args []string) bool {
			return name == "ln" && slices.Contains(args, "/bin/sh")
		},
	)
	if pos == nil {
		t.Fatal("expected to find command")
	}
	if pos.PrecedingCommands != "apt-get update" {
		t.Errorf("PrecedingCommands = %q, want %q", pos.PrecedingCommands, "apt-get update")
	}
	if pos.RemainingCommands != "echo done" {
		t.Errorf("RemainingCommands = %q, want %q", pos.RemainingCommands, "echo done")
	}
}

func TestFindCommandInChain_NoMatch(t *testing.T) {
	t.Parallel()
	pos := FindCommandInChain("apt-get update && echo hello", VariantBash, func(name string, args []string) bool {
		return name == "ln" && slices.Contains(args, "/bin/sh")
	})
	if pos != nil {
		t.Error("expected nil when no command matches")
	}
}

func TestFindCommandInChain_MatchesOnlyPredicatedArgs(t *testing.T) {
	t.Parallel()
	// ln without /bin/sh should not match
	pos := FindCommandInChain("ln -sf /bin/true /sbin/initctl", VariantBash, func(name string, args []string) bool {
		return name == "ln" && slices.Contains(args, "/bin/sh")
	})
	if pos != nil {
		t.Error("expected nil when ln does not target /bin/sh")
	}
}
