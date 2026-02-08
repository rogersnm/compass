# Compass

Markdown-native task and document tracking for your terminal.

Compass stores everything as `.md` files with YAML frontmatter. No database, no sync service, no lock-in. Your data lives in `~/.compass/` as plain text you can grep, version, and edit with any tool.

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
# Create a project
compass project create "My App"

# Set it as default so you don't need --project everywhere
compass project set-default PROJ-XXXXX

# Create an epic
compass task create "Authentication" --type epic

# Create tasks with dependencies
compass task create "Design login page" --epic TASK-AAAAA
compass task create "Implement OAuth" --epic TASK-AAAAA
compass task create "Write auth tests" --depends-on TASK-BBBBB,TASK-CCCCC

# See what's ready to work on
compass task ready --all

# Start working
compass task start TASK-BBBBB

# Pipe markdown content into tasks and docs
echo '# Login Page Spec

- Email + password
- OAuth with Google
- Rate limiting on failed attempts' | compass doc create "Login Spec"
```

## Concepts

**Projects** are top-level containers. Each project gets its own directory under `~/.compass/projects/`.

**Tasks** track work. They have a status (`open`, `in_progress`, `closed`) and can depend on other tasks. Dependencies form a DAG; compass validates acyclicity and uses topological sorting to determine what's ready.

**Epics** are tasks with `type: epic`. They group related tasks but cannot have dependencies themselves and cannot be depended on.

**Documents** store long-form markdown content (specs, design docs, notes) associated with a project.

**Blocked** is computed, not stored. A task is blocked if any of its dependencies are not yet closed.

## Commands

### Projects

```bash
compass project create "Name"       # Create a project
compass project list                 # List all projects
compass project show PROJ-XXXXX     # Show project details
compass project set-default PROJ-XXXXX  # Set default project
```

### Tasks

```bash
compass task create "Title" [--project P] [--type task|epic] [--epic E] [--depends-on T1,T2]
compass task list [--project P] [--status S] [--type T] [--epic E]
compass task show TASK-XXXXX
compass task update TASK-XXXXX [--title T] [--status S] [--depends-on T1,T2]
compass task edit TASK-XXXXX         # Open in $EDITOR
compass task start TASK-XXXXX        # Shortcut: set status to in_progress
compass task close TASK-XXXXX        # Shortcut: set status to closed
compass task delete TASK-XXXXX
compass task ready [--project P] [--all]
compass task graph [--project P]     # ASCII dependency graph
compass task checkout TASK-XXXXX     # Copy to .compass/ for local editing
compass task checkin TASK-XXXXX      # Write back to store, remove local copy
```

### Documents

```bash
compass doc create "Title" [--project P]
compass doc list [--project P]
compass doc show DOC-XXXXX
compass doc update DOC-XXXXX [--title T]
compass doc edit DOC-XXXXX
compass doc delete DOC-XXXXX
compass doc checkout DOC-XXXXX
compass doc checkin DOC-XXXXX
```

### Search

```bash
compass search "query" [--project P]    # Search across all entities
```

### Piping Content

Tasks and documents accept markdown body content via stdin:

```bash
echo '# Design Notes' | compass task create "Design review"
echo '# Updated spec' | compass doc update DOC-XXXXX
cat spec.md | compass doc create "API Specification"
```

## Checkout / Checkin

Compass stores data in `~/.compass/`, which is outside your working directory. AI coding tools and editors that operate on local files can use checkout/checkin to work with compass entities:

```bash
compass task checkout TASK-XXXXX     # Copies to .compass/TASK-XXXXX.md
# Edit .compass/TASK-XXXXX.md with any tool
compass task checkin TASK-XXXXX      # Validates, writes back, removes local copy
```

Checkin validates frontmatter and (for tasks) checks dependency constraints before writing back. If validation fails, the local file is preserved so you can fix it.

## Storage Layout

```
~/.compass/
├── config.yaml
└── projects/
    └── PROJ-XXXXX/
        ├── project.md
        ├── documents/
        │   └── DOC-XXXXX.md
        └── tasks/
            └── TASK-XXXXX.md
```

Each file is YAML frontmatter followed by a markdown body. You can edit them directly if you want.

## AI Tool Integration

Compass implements the [Model Tools Protocol](https://modeltoolsprotocol.io) (MTP) for discoverability by AI agents:

```bash
compass --mtp-describe    # JSON schema of all commands, args, and I/O types
```

This lets AI tools discover compass's capabilities, understand input/output formats, and generate correct commands without hardcoded knowledge.

## IDs

All entities use the format `PREFIX-HASH` where PREFIX is `PROJ`, `DOC`, or `TASK`, and HASH is 5 characters from a 30-character alphabet (`23456789ABCDEFGHJKMNPQRSTUVWXYZ`). Ambiguous characters (0/O, 1/I/L) are excluded to make IDs easier to read and communicate.

## License

MIT
