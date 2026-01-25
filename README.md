# tally

A fast, configurable linter for Dockerfiles and Containerfiles.

## Installation

### NPM

```bash
npm install -g @contino/tally
```

### PyPI

```bash
pip install tally-cli
```

### RubyGems

```bash
gem install tally-cli
```

### Go

```bash
go install github.com/tinovyatkin/tally@latest
```

### From Source

```bash
git clone https://github.com/tinovyatkin/tally.git
cd tally
go build .
```

## Usage

```bash
# Check a Dockerfile
tally check Dockerfile

# Check with max lines limit
tally check --max-lines 100 Dockerfile

# Output as JSON
tally check --format json Dockerfile

# Check multiple files
tally check Dockerfile.dev Dockerfile.prod
```

## Available Rules

| Rule | Description | Flag |
|------|-------------|------|
| `max-lines` | Enforce maximum number of lines | `--max-lines <n>` |

## Configuration

Currently, tally supports configuration via CLI flags. A configuration file format is planned for future releases.

## Output Formats

### Text (default)

```
Dockerfile:0: file has 150 lines, maximum allowed is 100 (max-lines)
```

### JSON

```json
[
  {
    "file": "Dockerfile",
    "lines": 150,
    "issues": [
      {
        "rule": "max-lines",
        "line": 0,
        "message": "file has 150 lines, maximum allowed is 100",
        "severity": "error"
      }
    ]
  }
]
```

## Contributing

See [CLAUDE.md](CLAUDE.md) for development guidelines.

## License

MIT
