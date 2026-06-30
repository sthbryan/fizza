# fizza

Single-binary kanban board manager, backed by SQLite, designed for both humans and LLMs.

- **One binary**, zero runtime dependencies (CGO-free, statically linked)
- **SQLite storage** at `~/.config/fizza/default.db` (single DB per user)
- **JSON output by default** so LLMs and other tools can consume it directly
- **MCP server** built-in: connect it to Claude Code, Cursor, or Continue in 30 seconds
- **Concurrent-safe**: fractional positions + WAL mode for multi-agent writes

## Install

### Homebrew (macOS/Linux)

```bash
brew install fizza/tap/fizza
```

### Download a release

Grab a binary for your platform from [the releases page](https://github.com/fizza/fizza/releases). Linux/macOS/Windows on amd64 and arm64, all statically linked.

### From source

```bash
go install github.com/fizza/fizza/cmd/fizza@latest
```

## Quickstart (30 seconds)

```bash
# Create a project (auto-seeds a default "main" board with todo/in_progress/done)
fizza project new alpha --desc "my first project"
fizza project new beta

# Set a default project so you don't have to pass --project on every call
fizza project set alpha

# Add tasks (uses the default project)
fizza task add "ship the demo" --board main --priority high
fizza task add "write the docs" --board main --due 2026-07-01

# Switch projects anytime
fizza project set beta
fizza task add "research competitor" --board main

# List
fizza task list --board main

# Move a task
fizza task move 1 done

# Show by ID prefix
fizza task show 1

# Human-friendly tables
fizza task list --board main --format pretty
```

## Connect to your coding agent (MCP)

Add fizza as an MCP server and any agent that supports the protocol gets access to 14 kanban tools.

### Claude Code

```bash
# After installing fizza
claude mcp add fizza -- fizza mcp
```

That's it. Claude Code will see `project_new`, `task_add`, `task_move`, etc. as native tools.

Verify:

```bash
claude mcp list
# fizza: fizza mcp - ✓ Connected
```

### Cursor

Add to `~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "fizza": {
      "command": "fizza",
      "args": ["mcp"]
    }
  }
}
```

### Continue (VS Code / JetBrains)

Add to `~/.continue/config.json`:

```json
{
  "experimental": {
    "modelContextProtocolServers": [
      {
        "name": "fizza",
        "transport": {
          "type": "stdio",
          "command": "fizza",
          "args": ["mcp"]
        }
      }
    ]
  }
}
```

### Manual / other agents

If your agent doesn't support MCP but can run subprocesses, point it at `fizza` directly. The CLI emits structured JSON to stdout for every command, so any agent that can parse JSON can drive it.

## Tools reference

| Tool | Purpose |
|---|---|
| `project_new` | Create a project (auto-seeds default board) |
| `project_list` | List all projects |
| `project_show` | Show a project by name |
| `project_delete` | Delete a project (cascades) |
| `board_create` | Create a board with optional custom columns |
| `board_list` | List boards in a project |
| `board_show` | Show a board with its columns |
| `board_delete` | Delete a board (blocked if it has tasks) |
| `task_add` | Create a task |
| `task_list` | List tasks in a board, optionally by column |
| `task_show` | Show a task by ID or numeric prefix |
| `task_move` | Move a task to a different column |
| `task_update` | Update one or more fields of a task |
| `task_delete` | Delete a task (cascades subtasks) |

All tools accept and return JSON. The MCP server enforces the same schema; invalid inputs are rejected before any DB write.

## Configuration

User config lives at `~/.config/fizza/config.json` (XDG-aware). Two fields:

| Key | Values | Effect |
|---|---|---|
| `mode` | `llm` (default) \| `human` | When `human`, output is rendered as tables without passing `--format` |
| `project` | project name | Default project for board/task commands. Override per-call by changing it |

```bash
fizza config show                   # show current config
fizza config set mode human         # human-readable output by default
fizza config set project alpha      # make `alpha` the default project
fizza config unset project          # clear the default (board/task commands will require it)
fizza config path                   # print the config file path
```

`fizza project set <name>` is a shortcut for `fizza config set project <name>` and validates that the project exists in the DB.

### Per-project config (`.fizza`)

For mono-repo or multi-project workflows, drop a `.fizza` file at the project root. It uses env-style `KEY=VALUE` lines and overrides the global config when fizza runs inside that directory or any subdirectory (search walks up to 5 levels and stops at the first `.git`, `$HOME`, or filesystem root).

```bash
# .fizza
PROJECT=myapp          # default project for this checkout
MODE=human             # default output mode (llm | human)
```

The closest `.fizza` to your cwd wins; keys not set fall back to the global config. Add `.fizza` to `.gitignore` if it's personal state; commit it if it's shared team config.

## CLI reference

```bash
fizza project new <name> [--desc "..."]
fizza project list
fizza project show <name>
fizza project delete <name> [--force]

fizza board create <name> [--columns "todo,in_progress,review,done"]
fizza board list
fizza board show <name>
fizza board delete <name> [--force]

fizza task add <title> --board <name> [--column <name>] [--desc "..."] [--priority low|medium|high|urgent] [--due 2026-07-01] [--parent <id>]
fizza task list --board <name> [--column <name>]
fizza task show <id>
fizza task move <id> <column> --board <name>
fizza task update <id> [--title "..."] [--desc "..."] [--priority ...] [--due ...] [--clear-due] [--parent <id>] [--clear-parent]
fizza task delete <id> [--force]

fizza mcp                                # run as MCP server on stdio

# Global flags
--format json|pretty                     # default json
--no-color                               # disable ANSI colors
```

### Output format

Every command returns a JSON envelope:

```json
{
  "ok": true,
  "data": { ... }
}
```

On error:

```json
{
  "ok": false,
  "error": {
    "code": "NOT_FOUND",
    "message": "task prefix \"99\""
  }
}
```

### Exit codes

| Code | Meaning |
|---|---|
| 0 | success |
| 1 | generic error |
| 2 | not found |
| 3 | validation error |
| 4 | duplicate |
| 5 | conflict (e.g. delete without `--force`) |

## Storage

Single database per user, located at `~/.config/fizza/default.db`. Honors `$XDG_CONFIG_HOME` (so `XDG_CONFIG_HOME=/tmp/foo` → `/tmp/foo/fizza/default.db`).

Schema is migrated automatically on first open. The DB file is yours — `sqlite3 ~/.config/fizza/default.db` works.

## Build from source

Requires Go 1.26+.

```bash
git clone https://github.com/fizza/fizza
cd fizza
make build         # ./fizza
make test          # unit + integration
make test-race     # with race detector
make mcp-test      # smoke-test the MCP server end-to-end
make install       # install to ~/.local/bin
```

## Releases

Releases are cut via GoReleaser on tag push:

```bash
git tag v0.1.0
git push --tags
```

GitHub Actions then builds for linux/darwin/windows × amd64/arm64, packages .deb/.rpm/.apk, and publishes to the GitHub release.

## License

MIT
