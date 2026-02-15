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

Compass is a markdown-native task/document tracking CLI supporting multiple stores (local filesystem and cloud instances). Local data lives in `~/.compass/` as `.md` files with YAML frontmatter. Cloud stores are accessed via REST API.

### Multi-store architecture

Compass supports multiple stores simultaneously. Each project is mapped to exactly one store via a cached lookup in `config.yaml`. Commands auto-route to the correct store based on the project key extracted from entity IDs.

- **Local store**: backed by `~/.compass/projects/`, enabled via `compass store add local`
- **Cloud stores**: identified by hostname (e.g. `compasscloud.io`), added via `compass store add <hostname>`
- **Store registry** (`internal/store/registry.go`): routes commands to stores via `ForProject()`/`ForEntity()` with cache-hit/miss/stale logic
- **Project cache** (`config.yaml` `projects` map): `projectKey -> storeName`, populated by `store fetch` or lazily on first access

### Config format (v2)

```yaml
version: 2
default_store: local
local_enabled: true
stores:
  compasscloud.io:
    hostname: compasscloud.io
    api_key: cpk_personal
  work:
    hostname: compasscloud.io
    api_key: cpk_work_org
projects:
  AUTH: local
  API: compasscloud.io
  WORK: work
```

Store names (map keys) are user-chosen; `hostname` is always explicit. Old configs without `hostname` get it backfilled from the map key on load. Multiple stores can point to the same hostname (e.g. different orgs/accounts). V1 configs (no `version` field) are auto-migrated on first load.

### Storage layout (local store)

```
~/.compass/
├── config.yaml                    # v2 multi-store config
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

Epics have no status. They must not have a `status` field in frontmatter, display "N/A" in listings, and are excluded from status filtering. Status-changing commands (`task start`, `task close`, `task update --status`) are rejected on epics.

### Project resolution

Commands that need a project call `resolveProject()` which checks in order:

1. `--project` / `-P` flag
2. `.compass-project` file in cwd or any ancestor (via `internal/repofile`)
3. Error if neither found

### Package responsibilities

- `cmd/` - Cobra commands. Global state (`reg`, `cfg`, `dataDir`) is set in `PersistentPreRunE`. Uses `storeForProject()`/`storeForEntity()` helpers to route to the correct store. Commands that don't need stores (`go`, `store`, `config`) are exempted from the store-check in `PersistentPreRunE`.
- `internal/store/` - `Store` interface, `Local` filesystem implementation, `CloudStore` REST client, and `Registry` for multi-store routing. `ResolveEntityPath()` computes paths directly from the ID (no scanning).
- `internal/model/` - Structs with `Validate()` methods. No I/O.
- `internal/dag/` - Graph construction from `[]*model.Task`, cycle detection, topological sort, ASCII rendering.
- `internal/markdown/` - Generic `Parse[T]()` / `Marshal()` for frontmatter round-tripping, glamour rendering, lipgloss table helpers (`RenderTaskTable`, `RenderProjectTable`, etc.).
- `internal/config/` - V2 multi-store config with migration from v1. `CloudStoreConfig` type with `Hostname` field and `URL()` method.
- `internal/id/` - ID generation and parsing: `GenerateKey()`, `NewTaskID()`, `NewDocID()`, `Parse()`, `TypeOf()`, `ProjectKeyFrom()`.
- `internal/repofile/` - `.compass-project` file discovery. `Find()` walks up directories; `Write()` / `Read()` manage the file.
- `internal/editor/` - Opens files in `$EDITOR` / `$VISUAL` / `vi`.

### MTP integration

`cmd/root.go` registers command annotations (stdin/stdout descriptors, examples) via `github.com/modeltoolsprotocol/go-sdk`. The `--mtp-describe` flag emits a JSON schema of all commands.

## Testing

Two test suites in `cmd/`:

- **`cmd_test.go`** (local store): `setupEnv(t)` creates a temp dir with v2 config and a fresh registry. `run(t, args...)` calls `rootCmd.SetArgs()` + `Execute()`.
- **`cloud_cmd_test.go`** (cloud store): `setupCloudEnv(t)` starts an `httptest.Server` backed by `fakeAPI` (in-memory cloud API implementation). Tests exercise the same commands through real HTTP.

Both suites use `setupEnv`/`setupCloudEnv` before each test to reset Cobra flag state.

## Key gotchas

- **Cobra flag persistence:** Flag values persist across `Execute()` calls in the same process. CLI tests must explicitly pass all flags and reset global state via `setupEnv()`.
- **`adrg/frontmatter`** does NOT error on missing frontmatter; it returns an empty struct.
- **stdin detection:** Uses `os.ModeNamedPipe` check (not `ModeCharDevice`), because the latter fails in piped environments like Claude Code.
- **Version injection:** `cmd.version` is a `var` defaulting to `"dev"`, stamped by GoReleaser via ldflags.
- **PersistentPreRunE skip list:** Commands that don't need store infrastructure (`go`, `store`, `config`) must be exempted in `root.go`'s `PersistentPreRunE`; otherwise they trigger the first-run setup prompt.

## Release

Tag-driven via GoReleaser + GitHub Actions. Cross-compiles for linux/darwin/windows, amd64/arm64. Publishes to GitHub Releases and `rogersnm/homebrew-tap`.

```bash
git tag v0.x.0
git push origin v0.x.0
```

**Never delete and retag an existing version.** If a tag has been pushed, bump to the next version instead. Force-pushing tags breaks GoReleaser, Homebrew caches, and anyone who already pulled the release.
