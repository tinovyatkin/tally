# CI/CD

tally is designed to run fast in CI without requiring Docker or a daemon.

## Common patterns

Fail the build when violations are found:

```bash
tally lint .
```

Use GitHub Actions annotations:

```bash
tally lint --format github-actions .
```

Generate SARIF output:

```bash
tally lint --format sarif --output tally.sarif .
```

## Tips

- Use `--fail-level` to control what severities fail CI (e.g. `warning` vs `error`).
- Use `--exclude` to skip generated/vendor trees.
- Use repo config (`.tally.toml`) to keep CI and local runs consistent.

See also:

- [Configuration](./configuration.md)
- [Output Formats](./output-formats.md)
