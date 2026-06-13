<!-- file: .github/copilot-instructions.md -->
<!-- version: 2.4.0 -->
<!-- guid: 4d5e6f7a-8b9c-0d1e-2f3a-4b5c6d7e8f9a -->
<!-- last-edited: 2026-06-13 -->

# subtitle-manager — Additional Context

Org-wide coding standards (file headers, language rules, commit format) are at
**https://github.com/falkcorp/.github** and apply automatically to this repo.

For full project context: **CLAUDE.md** at the repo root.

## Project overview

Full-stack subtitle management tool — Go backend + React/TypeScript frontend.
The Go binary embeds the React frontend via `//go:embed`. Frontend must be built first.

## Key directories

| Directory | Purpose |
|-----------|---------|
| `cmd/` | CLI subcommands (fetch, merge, extract, etc.) |
| `pkg/` | Library packages (engine, database, audio, etc.) |
| `proto/` | Protobuf definitions |
| `webui/` | React/TypeScript frontend |
| `bin/` | Compiled binaries |

## Build commands

```bash
make build        # Full build: webui + go generate + binary
make binary       # Go binary only
make test         # Go tests
make test-all     # Go + WebUI tests
make webui-dev    # Vite dev server (frontend only)
```
