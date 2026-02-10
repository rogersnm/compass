# Compass

Markdown-native task and document tracking for your terminal.

Compass stores everything as `.md` files with YAML frontmatter. No database, no lock-in. Your data lives in `~/.compass/` as plain text you can grep, version, and edit with any tool. Connect multiple stores (local filesystem and cloud instances) and compass routes commands to the right one automatically.

## Install

**Homebrew:**

```bash
brew install rogersnm/tap/compass
```

**From source:**

```bash
go install github.com/rogersnm/compass@latest
```

## Quick Start

```bash
# Add a local store (first time only)
compass store add local

# Create a project (auto-generates key "MYAP" from name)
compass project create "My App"

# Or specify a key explicitly
compass project create "My App" --key APP

# Link this repo to the project (writes .compass-project in cwd)
compass repo init APP

# Or set a global default instead
compass project set-default APP

# Create an epic
compass task create "Authentication" --type epic

# Create tasks with dependencies
compass task create "Design login page" --parent-epic APP-TAAAAA
compass task create "Implement OAuth" --parent-epic APP-TAAAAA
compass task create "Write auth tests" --depends-on APP-TBBBBB,APP-TCCCCC

# See what's ready to work on
compass task ready --all

# Start working
compass task start APP-TBBBBB

# Set priority (P0 = critical, P3 = low)
compass task create "Fix login crash" --priority 0

# Pipe markdown content into tasks and docs
echo '# Login Page Spec

- Email + password
- OAuth with Google
- Rate limiting on failed attempts' | compass doc create "Login Spec"
```

## Concepts

**Projects** are top-level containers. Each project has a key (2-5 uppercase alphanumeric chars) that becomes part of every entity ID. Keys are auto-generated from the project name or set explicitly with `--key`.

**Tasks** track work. They have a status (`open`, `in_progress`, `closed`), an optional priority (P0-P3), and can depend on other tasks. Dependencies form a DAG; compass validates acyclicity and uses topological sorting to determine what's ready.

**Epics** are tasks with `type: epic`. They group related tasks but cannot have dependencies themselves and cannot be depended on.

**Documents** store long-form markdown content (specs, design docs, notes) associated with a project.

**Blocked** is computed, not stored. A task is blocked if any of its dependencies are not yet closed.

## IDs

IDs are project-key-based, so you can tell which project an entity belongs to at a glance:

| Entity   | Format      | Example        |
|----------|-------------|----------------|
| Project  | `KEY`       | `AUTH`         |
| Task     | `KEY-THASH` | `AUTH-TA7K2P`  |
| Document | `KEY-DHASH` | `AUTH-DA7K2P`  |

Keys are auto-generated from the first 4 alpha characters of the project name (uppercased). On collision, a digit is appended: `AUTH`, `AUTH2`, `AUTH3`, etc. The hash portion uses a 30-character alphabet (`23456789ABCDEFGHJKMNPQRSTUVWXYZ`) with ambiguous characters (0/O, 1/I/L) excluded.

## Commands

### Projects

```bash
compass project create "Name" [--key K] [--store S]  # Create a project
compass project list                                  # List all projects (from cache)
compass project show AUTH                             # Show project details
compass project set-store AUTH compasscloud.io        # Reassign project to a different store
```

### Tasks

```bash
compass task create "Title" [--project P] [--type task|epic] [--parent-epic E] [--depends-on T1,T2] [--priority 0-3]
compass task list [--project P] [--status S] [--type T] [--parent-epic E]
compass task show AUTH-TXXXXX
compass task update AUTH-TXXXXX [--title T] [--status S] [--depends-on T1,T2] [--priority 0-3]
compass task edit AUTH-TXXXXX             # Open in $EDITOR
compass task start AUTH-TXXXXX            # Shortcut: set status to in_progress
compass task close AUTH-TXXXXX            # Shortcut: set status to closed
compass task delete AUTH-TXXXXX
compass task ready [--project P] [--all]
compass task graph [--project P]          # ASCII dependency graph
compass task download AUTH-TXXXXX         # Copy to .compass/ for local editing
compass task upload AUTH-TXXXXX           # Write back to store, remove local copy
```

### Documents

```bash
compass doc create "Title" [--project P]
compass doc list [--project P]
compass doc show AUTH-DXXXXX
compass doc update AUTH-DXXXXX [--title T]
compass doc edit AUTH-DXXXXX
compass doc delete AUTH-DXXXXX
compass doc download AUTH-DXXXXX
compass doc upload AUTH-DXXXXX
```

### Repo Linking

```bash
compass repo init [PROJECT-ID]          # Link cwd to a project (writes .compass-project)
compass repo show                       # Show current repo-project link
compass repo unlink                     # Remove .compass-project from cwd
```

### Stores

Compass supports multiple stores simultaneously. Each project lives on exactly one store; commands auto-route based on a cached project-to-store mapping.

```bash
compass store add local                          # Enable local filesystem store
compass store add compasscloud.io                # Add a cloud store (device flow login)
compass store add compasscloud.io --api-key KEY  # Add with API key (CI/non-interactive)
compass store list                               # List configured stores
compass store set-default local                  # Set default store for new projects
compass store fetch                              # Fetch and cache projects from all stores
compass store fetch --store compasscloud.io      # Fetch from one store
compass store fetch --all                        # Non-interactive, add all projects
compass store remove compasscloud.io             # Remove a store (prompts if projects mapped)
```

### Search

```bash
compass search "query" [--project P]    # Search across all entities
```

### Piping Content

Tasks and documents accept markdown body content via stdin:

```bash
echo '# Design Notes' | compass task create "Design review"
echo '# Updated spec' | compass doc update AUTH-DXXXXX
cat spec.md | compass doc create "API Specification"
```

## Project Resolution

Commands that need a project resolve it in this order:

1. `--project` / `-P` flag (explicit, highest priority)
2. `.compass-project` file in the current directory or any ancestor

The `.compass-project` file is a single-line text file containing a project key (like `.nvmrc` or `.node-version`). Run `compass repo init` to create one.

## Download / Upload

Compass stores data in `~/.compass/`, which is outside your working directory. AI coding tools and editors that operate on local files can use download/upload to work with compass entities:

```bash
compass task download AUTH-TXXXXX     # Copies to .compass/AUTH-TXXXXX.md
# Edit .compass/AUTH-TXXXXX.md with any tool
compass task upload AUTH-TXXXXX       # Validates, writes back, removes local copy
```

Upload validates frontmatter and (for tasks) checks dependency constraints before writing back. If validation fails, the local file is preserved so you can fix it.

## Storage Layout

```
~/.compass/
├── config.yaml          # Multi-store config (v2)
└── projects/            # Local store data
    └── AUTH/
        ├── project.md
        ├── documents/
        │   └── AUTH-DXXXXX.md
        └── tasks/
            └── AUTH-TXXXXX.md
```

Local store files are YAML frontmatter followed by a markdown body. You can edit them directly if you want. Cloud store data lives on the remote server and is accessed via API.

## AI Tool Integration

Compass implements the [Model Tools Protocol](https://modeltoolsprotocol.io) (MTP) for discoverability by AI agents:

```bash
compass --mtp-describe    # JSON schema of all commands, args, and I/O types
```

This lets AI tools discover compass's capabilities, understand input/output formats, and generate correct commands without hardcoded knowledge.

## License

MIT
