# Auto-Fix

tally can apply fixes automatically. Fixes are designed to be:

- **Atomic** (a fix applies fully or not at all)
- **Conflict-aware** (overlapping edits are skipped)
- **Configurable** (per-rule fix modes)

## Basic usage

Apply safe fixes:

```bash
tally lint --fix Dockerfile
```

Apply unsafe fixes too (includes AI fixes when enabled):

```bash
tally lint --fix --fix-unsafe Dockerfile
```

Limit fixes to specific rules:

```bash
tally lint --fix --fix-unsafe --fix-rule hadolint/DL3008 --fix-rule tally/prefer-copy-heredoc Dockerfile
```

## Per-rule fix modes

You can control when fixes are allowed in `.tally.toml`:

```toml
[rules.tally.prefer-copy-heredoc]
fix = "always"        # default behavior

[rules.tally.prefer-multi-stage-build]
fix = "explicit"      # only when --fix-rule includes this rule
```

Valid values:

- `always` (default)
- `never`
- `explicit` (requires `--fix-rule`)
- `unsafe-only` (requires `--fix-unsafe`)

## AI AutoFix

Some fixes are too complex to implement deterministically. For those, tally supports an opt-in AI resolver via ACP.

See:

- [AI AutoFix (ACP/MCP)](./ai-autofix-acp.md)
