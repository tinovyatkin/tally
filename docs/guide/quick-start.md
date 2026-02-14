# Quick Start

Lint a Dockerfile:

```bash
tally lint Dockerfile
```

Lint all Dockerfiles in the current repo (recursive discovery):

```bash
tally lint .
```

Apply safe fixes automatically:

```bash
tally lint --fix Dockerfile
```

Enable context-aware checks (e.g. `.dockerignore` / `.containerignore` interactions):

```bash
tally lint --context . Dockerfile
```

Next:

- [Configuration](./configuration.md)
- [Auto-Fix](./auto-fix.md)
- [Output Formats](./output-formats.md)
