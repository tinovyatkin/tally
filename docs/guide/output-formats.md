# Output Formats

tally supports multiple output formats so it can fit into both terminals and automation.

## text (default)

Human-readable output with source snippets:

```bash
tally lint Dockerfile
```

## json

Machine-readable output:

```bash
tally lint --format json Dockerfile
```

## sarif

Static analysis output for code scanning tools:

```bash
tally lint --format sarif --output tally.sarif .
```

## github-actions

Emits GitHub Actions annotations:

```bash
tally lint --format github-actions .
```

## markdown

Concise markdown tables (useful for AI tooling and reports):

```bash
tally lint --format markdown Dockerfile
```
