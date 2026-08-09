# Tally - Dockerfile Linter, Formatter & Modernizer

**The Dockerfile linter that understands your stack.**

Tally brings BuildKit-native Dockerfile intelligence to VS Code: stack-aware
PHP, Ruby, PowerShell, and shell rules; security checks; modern Dockerfile
guidance; formatting; and safe auto-fixes. Marketplace builds include the
native `tally` binary, so Docker Desktop and a separate CLI installation are
not required.

![Tally diagnostics in VS Code](https://raw.githubusercontent.com/wharflab/tally/main/_integrations/vscode-tally/assets/marketplace-diagnostics.png)

## Why Tally

- **Stack-aware rules** for PHP, Ruby, PowerShell, Windows containers, package
  ecosystems, and embedded shell.
- **Modern Dockerfile analysis** for heredocs, BuildKit cache mounts,
  `COPY --link`, modern `ADD` sources, multi-stage builds, and current security
  practices.
- **Real language understanding** through embedded ShellCheck and a PowerShell
  parser instead of treating every `RUN` instruction as opaque text.
- **Build context awareness** across stages, effective `ENV` and `SHELL`
  values, registries, `.dockerignore`, Docker Compose, and Buildx Bake.
- **Fixes, not only findings** with Quick Fix, iterative Fix All, document
  formatting, and opt-in AI AutoFix for changes that require deeper reasoning.
- **BuildKit and Hadolint compatibility** for teams migrating existing checks
  and policy while adopting Tally's richer rule families.

## See It In Action

Tally publishes diagnostics to the Problems panel and provides contextual code
actions directly in the editor:

![Tally Quick Fix actions](https://raw.githubusercontent.com/wharflab/tally/main/_integrations/vscode-tally/assets/marketplace-quick-fix.png)

Safe fixes can be applied together without leaving VS Code:

![Dockerfile after Tally Fix All](https://raw.githubusercontent.com/wharflab/tally/main/_integrations/vscode-tally/assets/marketplace-fixed.png)

## Install

Install `wharflab.tally` from the Visual Studio Marketplace, or run:

```bash
code --install-extension wharflab.tally
```

Then open a `Dockerfile` or `Containerfile`. Diagnostics appear automatically.
Use `Tally: Fix all auto-fixable issues` to apply safe fixes, or
`Tally: Configure as default formatter for Dockerfile` to enable formatting on
save.

## Commands

- `Tally: Fix all auto-fixable issues`: iteratively applies safe fixes. Set
  `tally.fixUnsafe=true` to include unsafe fixes.
- `Tally: Configure as default formatter for Dockerfile`: configures Tally and
  format on save at the workspace or user level.
- `Tally: Restart server`
- `Tally: Show output`
- `Tally: Show LSP trace`

## Settings

- `tally.enable`: enable or disable the language server.
- `tally.path`: explicit paths to a `tally` executable; the first existing path
  wins.
- `tally.importStrategy`: resolve Tally from the environment or require the
  bundled binary.
- `tally.configuration`: inline configuration merged with `.tally.toml` or
  `tally.toml`.
- `tally.configurationPreference`: choose how editor and filesystem
  configuration are merged.
- `tally.fixUnsafe`: allow manual Fix All to apply unsafe fixes, including
  configured AI AutoFix rules.
- `tally.trace.server`: control LSP protocol tracing.

## Formatter Setup

The `Tally: Configure as default formatter for Dockerfile` command writes this
configuration for you:

```jsonc
{
  "[dockerfile]": {
    "editor.defaultFormatter": "wharflab.tally",
    "editor.formatOnSave": true,
    "editor.formatOnSaveMode": "file",
    "editor.codeActionsOnSave": {
      "source.fixAll.tally": "explicit"
    }
  }
}
```

## Binary Resolution

Marketplace packages include a platform-specific binary. By default, Tally
first respects explicitly configured and project-local installations, then
falls back to the bundled binary. Set `tally.importStrategy` to `useBundled` to
require the bundled version.

Python virtual environments are also discovered. When Microsoft's Python
Environments extension is installed, Tally follows the environment selected
for the current workspace; otherwise it checks common `.venv` and `venv`
locations.

## Documentation

- Rules reference: <https://tally.wharflab.com/rules/overview>
- Configuration: <https://tally.wharflab.com/guides/configuration>
- Auto-fix: <https://tally.wharflab.com/guides/auto-fix>
- AI AutoFix: <https://tally.wharflab.com/guides/ai-autofix>
- Build invocations: <https://tally.wharflab.com/guides/build-invocations>
