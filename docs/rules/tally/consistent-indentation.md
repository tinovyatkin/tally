# tally/consistent-indentation

Enforces consistent indentation for Dockerfile build stages.

| Property | Value |
|----------|-------|
| Severity | Style |
| Category | Style |
| Default | Off (experimental) |
| Auto-fix | Yes (safe) |

## Description

Enforces consistent indentation to visually separate build stages in multi-stage Dockerfiles.

**Behavior depends on the number of stages:**

- **Multi-stage** (2+ FROM instructions): Commands within each stage must be indented. FROM lines remain at column 0.
- **Single-stage** (1 FROM instruction): All indentation is removed — tabs, spaces, or any mix. Since there is no stage structure to communicate,
  indenting commands adds noise. The auto-fix strips all leading whitespace from every instruction.

Tabs are the default indent character because they work well with heredoc `<<-` tab stripping, which removes leading tabs from heredoc content.

### Multi-stage (indentation required)

```dockerfile
FROM golang:1.23 AS builder
	WORKDIR /src
	COPY . .
	RUN go build -o /app

FROM alpine:3.20
	COPY --from=builder /app /usr/local/bin/app
	ENTRYPOINT ["app"]
```

### Single-stage (no indentation)

```dockerfile
FROM alpine:3.20
RUN apk add --no-cache curl
COPY . /app
CMD ["./app"]
```

## Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `indent` | string | `"tab"` | Indentation character: `"tab"` or `"space"` |
| `indent-width` | integer | `1` | Number of indent characters per level (1-8) |

### Why tabs?

Tabs are recommended because Docker heredoc syntax (`<<-`) strips leading tabs:

```dockerfile
FROM alpine:3.20
	COPY <<-EOF /etc/config
		key=value
		other=setting
	EOF
```

With spaces, `<<-` cannot strip indentation.

## Examples

### Bad (multi-stage without indentation)

```dockerfile
FROM golang:1.23 AS builder
WORKDIR /src
RUN go build -o /app
FROM alpine:3.20
COPY --from=builder /app /app
```

### Good (multi-stage with tab indentation)

```dockerfile
FROM golang:1.23 AS builder
	WORKDIR /src
	RUN go build -o /app
FROM alpine:3.20
	COPY --from=builder /app /app
```

### Bad (single-stage with tab indentation)

In a single-stage Dockerfile, indentation is unnecessary and will be removed by `--fix`:

```dockerfile
# Before (violation: unexpected indentation)
FROM alpine:3.20
	RUN apk add curl
	COPY . /app

# After --fix (indentation removed)
FROM alpine:3.20
RUN apk add curl
COPY . /app
```

### Bad (single-stage with space indentation)

Spaces are stripped the same way as tabs:

```dockerfile
# Before (violation: unexpected indentation)
FROM alpine:3.20
    RUN apk add curl
    COPY . /app

# After --fix (indentation removed)
FROM alpine:3.20
RUN apk add curl
COPY . /app
```

### Good (single-stage without indentation)

```dockerfile
FROM alpine:3.20
RUN apk add curl
COPY . /app
```

## Configuration

Enable the rule and configure indentation style:

```toml
[rules.tally.consistent-indentation]
severity = "style"
indent = "tab"
indent-width = 1
```

For 4-space indentation:

```toml
[rules.tally.consistent-indentation]
severity = "style"
indent = "space"
indent-width = 4
```

## Auto-fix

This rule provides safe auto-fixes that adjust indentation:

- **Multi-stage**: Adds the configured indentation to commands within stages
- **Single-stage**: Removes all leading whitespace (tabs and spaces) from commands
- **Style correction**: Replaces wrong indent characters (e.g., spaces to tabs)
- **Heredoc `<<-` conversion**: When tab indentation is applied to a heredoc instruction (`RUN <<EOF`, `COPY <<EOF`), the fix also converts `<<` to
  `<<-` so that BuildKit strips the leading tabs from the heredoc body

```bash
tally check --fix Dockerfile
```
