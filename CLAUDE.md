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
├── config.yaml                    # default_project (stores project key)
└── projects/
    └── AUTH/                      # project key (2-5 uppercase alphanumeric)
        ├── project.md
        ├── documents/AUTH-DXXXXX.md
        └── tasks/AUTH-TXXXXX.md   # both task and epic types
```

Each `.md` file has YAML frontmatter (parsed by `adrg/frontmatter`) followed by a markdown body. The `internal/markdown` package provides generic `Parse[T]()` and `Marshal()` for round-tripping.

### Entity model

Three entity types: Project, Document, Task. Epics are tasks with `type: epic`.

ID format is project-key-based:
- **Project**: bare key (e.g. `AUTH`, `AUTH2`, `API`)
- **Task**: `KEY-THASH` (e.g. `AUTH-TABCDE`)
- **Document**: `KEY-DHASH` (e.g. `AUTH-DABCDE`)

Keys are 2-5 uppercase alphanumeric chars. Hash is 5 chars from charset `23456789ABCDEFGHJKMNPQRSTUVWXYZ` (no ambiguous 0/O/1/I/L). Keys are auto-generated from the project name (first 4 alpha chars, uppercased) with collision handling (AUTH, AUTH2, AUTH3...) or explicitly provided via `--key`. The `internal/id` package handles generation and parsing.

Tasks have a DAG of dependencies via `depends_on`. Epic-type tasks cannot have dependencies and cannot be depended on. The `internal/dag` package validates acyclicity (DFS) and provides topological sorting (Kahn's algorithm) for the `task ready` command.

"Blocked" is computed, not stored: a task is blocked if any dependency is not closed.

Epic status is computed, not stored: derived from child tasks via `model.ComputeEpicStatus()`. No children or all closed = `closed`, all open = `open`, any `in_progress` = `in_progress`. Epics must not have a `status` field in frontmatter. Status-changing commands (`task start`, `task close`, `task update --status`) are rejected on epics.

### Package responsibilities

- `cmd/` - Cobra commands. Global state (`st`, `cfg`, `dataDir`) is set in `PersistentPreRunE`.
- `internal/store/` - All file I/O. `ResolveEntityPath()` computes paths directly from the ID (no scanning).
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

**Never delete and retag an existing version.** If a tag has been pushed, bump to the next version instead. Force-pushing tags breaks GoReleaser, Homebrew caches, and anyone who already pulled the release.
