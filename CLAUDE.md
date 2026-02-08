# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test

```bash
go build ./...              # Build all packages
go test ./...               # Run all tests
go test ./internal/store/   # Run tests for a single package
go test -run TestReadyTasks ./internal/store/  # Run a single test
go build -o compass .       # Build the binary
```

No linter is configured. No Makefile. GoReleaser handles release builds.

## Architecture

Compass is a markdown-native task/document tracking CLI. Data lives in `~/.compass/` as `.md` files with YAML frontmatter. There is no database; the filesystem is the source of truth.

### Storage layout

```
~/.compass/
├── config.yaml                    # default_project
└── projects/
    └── PROJ-XXXXX/
        ├── project.md
        ├── documents/DOC-XXXXX.md
        └── tasks/TASK-XXXXX.md    # both task and epic types
```

Each `.md` file has YAML frontmatter (parsed by `adrg/frontmatter`) followed by a markdown body. The `internal/markdown` package provides generic `Parse[T]()` and `Marshal()` for round-tripping.

### Entity model

Three entity types: Project, Document, Task. Epics are tasks with `type: epic`. All IDs use the format `PREFIX-HASH` where HASH is 5 chars from charset `23456789ABCDEFGHJKMNPQRSTUVWXYZ` (no ambiguous 0/O/1/I/L). The `internal/id` package handles generation and parsing.

Tasks have a DAG of dependencies via `depends_on`. Epic-type tasks cannot have dependencies and cannot be depended on. The `internal/dag` package validates acyclicity (DFS) and provides topological sorting (Kahn's algorithm) for the `task ready` command.

"Blocked" is computed, not stored: a task is blocked if any dependency is not closed.

### Package responsibilities

- `cmd/` - Cobra commands. Global state (`st`, `cfg`, `dataDir`) is set in `PersistentPreRunE`.
- `internal/store/` - All file I/O. `ResolveEntityPath()` scans project dirs to find an entity by ID.
- `internal/model/` - Structs with `Validate()` methods. No I/O.
- `internal/dag/` - Graph construction from `[]*model.Task`, cycle detection, topological sort, ASCII rendering.
- `internal/markdown/` - Frontmatter parse/marshal, glamour rendering, lipgloss tables.
- `internal/config/` - YAML config for default project.

### MTP integration

`cmd/root.go` registers command annotations (stdin/stdout descriptors, examples) via `github.com/modeltoolsprotocol/go-sdk`. The `--mtp-describe` flag emits a JSON schema of all commands.

## Key gotchas

- **Cobra flag persistence:** Flag values persist across `Execute()` calls in the same process. CLI tests must explicitly pass all flags and reset global state via `setupEnv()`.
- **`adrg/frontmatter`** does NOT error on missing frontmatter; it returns an empty struct.
- **stdin detection:** Uses `os.ModeNamedPipe` check (not `ModeCharDevice`), because the latter fails in piped environments like Claude Code.
- **Version injection:** `cmd.version` is a `var` defaulting to `"dev"`, stamped by GoReleaser via ldflags.

## Release

Tag-driven via GoReleaser + GitHub Actions. Cross-compiles for linux/darwin/windows, amd64/arm64. Publishes to GitHub Releases and `rogersnm/homebrew-tap`.

```bash
git tag v0.x.0
git push origin v0.x.0
```
