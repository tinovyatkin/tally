# tally/prefer-add-unpack

Prefer `ADD --unpack` for downloading and extracting remote archives.

| Property | Value |
|----------|-------|
| Severity | Info |
| Category | Performance |
| Default | Enabled |
| Auto-fix | Yes (`--fix --fix-unsafe`) |

## Description

Flags `RUN` instructions that download a remote archive with `curl`/`wget` and extract it with `tar`, suggesting `ADD --unpack <url> <dest>` instead.

`ADD --unpack` is a BuildKit feature that downloads and extracts a remote archive in a single layer, reducing image size and build complexity.

## Detected Patterns

1. **Pipe pattern**: `curl -fsSL <url> | tar -xz -C /dest`
2. **Download-then-extract**: `curl -o /tmp/app.tar.gz <url> && tar -xf /tmp/app.tar.gz -C /dest`
3. **wget variants**: Same patterns with `wget` instead of `curl`

The rule checks that the URL has a recognized archive extension (`.tar.gz`, `.tgz`, `.tar.bz2`, `.tar.xz`, etc.) and that a tar extraction command is
present in the same `RUN` instruction.

## Examples

### Before (violation)

```dockerfile
FROM ubuntu:22.04
RUN curl -fsSL https://go.dev/dl/go1.22.0.linux-amd64.tar.gz | tar -xz -C /usr/local

RUN wget -O /tmp/node.tar.xz https://nodejs.org/dist/v20.11.0/node-v20.11.0-linux-x64.tar.xz && \
    tar -xJf /tmp/node.tar.xz -C /usr/local --strip-components=1
```

### After (fixed with --fix --fix-unsafe)

```dockerfile
FROM ubuntu:22.04
ADD --unpack https://go.dev/dl/go1.22.0.linux-amd64.tar.gz /usr/local

ADD --unpack https://nodejs.org/dist/v20.11.0/node-v20.11.0-linux-x64.tar.xz /usr/local
```

## Auto-fix Conditions

The auto-fix is only emitted when the `RUN` instruction contains **only** download and extraction commands (curl/wget + tar). If additional commands
are present (e.g. `chmod`, `rm`, `mv`), the violation is still reported but no fix is suggested, since those commands would be lost.

The tar destination is extracted from `-C`, `--directory=`, or `--directory` flags. If no destination is specified, `/` is used as the default.

## Limitations

- Only detects `curl` and `wget` as download commands
- Only emits auto-fix when `tar` extraction is present (single-file decompressors like `gunzip`/ `bunzip2` are detected as violations but not
  auto-fixed, since `ADD --unpack` only unpacks tar archives)
- Skips non-POSIX shells (e.g. PowerShell stages)
- URL must have a recognized archive file extension

## Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | boolean | true | Enable or disable the rule |

## Configuration

```toml
[rules.tally.prefer-add-unpack]
enabled = true
```
