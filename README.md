# fizza

Local kanban for humans and coding agents. One static binary, SQLite storage, structured CLI output, and a built-in [MCP](https://modelcontextprotocol.io/) server.

## Features

- **Single binary** — CGO-free, statically linked; no runtime dependencies
- **SQLite backend** — per-user database at `~/.config/fizza/default.db` (XDG)
- **Agent-ready I/O** — JSON (default) and compact TOON output; stable exit codes
- **MCP server** — 15 tools over stdio for Claude Code, Cursor, Continue, and others
- **Concurrent writes** — WAL mode, busy timeout, fractional task positions
- **Projects → boards → columns → tasks** — tags, WIP limits, subtasks, audit events

## Install

### Homebrew

```bash
brew install fizza/tap/fizza
```

### Prebuilt binaries

Download for your platform from [GitHub Releases](https://github.com/fizza/fizza/releases)  
(Linux, macOS, Windows — amd64 and arm64).

### Go

```bash
go install github.com/fizza/fizza/cmd/fizza@latest
```

## Quick start

```bash
fizza project new myapp
fizza config set project myapp
fizza config set board main

fizza task add "Ship MCP integration"
fizza task add "Write docs" --priority high
fizza task list
fizza task move 1 done
```

Human-readable tables:

```bash
fizza config set mode human
# or per command:
fizza task list --format pretty
```

## MCP (coding agents)

```bash
fizza mcp   # stdio MCP server
```

### Claude Code

```bash
claude mcp add fizza -- fizza mcp
claude mcp list
```

### Cursor

`~/.cursor/mcp.json`:

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

### Continue

`~/.continue/config.json`:

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

Agents without MCP can shell out to `fizza` and parse JSON from stdout.

### Tools

| Tool | Description |
|------|-------------|
| `project_new` | Create project (seeds board `main`: todo / in_progress / done) |
| `project_list` | List projects, or one project if `name` is set |
| `project_delete` | Delete project and children (`force=true`) |
| `board_create` | Create board; optional custom columns |
| `board_list` | List boards; with `name`, returns board and columns |
| `board_snapshot` | Full board: columns with tasks in order |
| `board_delete` | Delete board if empty (`force=true`) |
| `task_add` | Create task (first column by default; honors WIP) |
| `task_list` | List or filter tasks; `id` returns a single task |
| `task_move` | Move to column; optional `before` for reorder |
| `task_update` | Patch fields; `add_tags` / `remove_tags` for labels |
| `task_delete` | Delete task and subtasks (`force=true`) |
| `tag_add` | Create tag in a project |
| `tag_list` | List tags |
| `tag_delete` | Delete tag (`force=true`) |

`project` and `board` arguments default from user config (or a local `.fizza` file) when omitted. Destructive tools require `force=true`.

## Configuration

Global config: `~/.config/fizza/config.json`

| Key | Values | Description |
|-----|--------|-------------|
| `mode` | `llm` (default), `human` | Default output style |
| `project` | name | Default project for board/task commands |
| `board` | name | Default board when a project has several |

```bash
fizza config show
fizza config set mode human
fizza config set project alpha
fizza config set board main
fizza config unset project
fizza config path
```

### Project-local config (`.fizza`)

Place a `.fizza` file in a repository root. Closest match wins (walks up at most 5 levels; stops at `.git`, `$HOME`, or filesystem root). Unset keys fall back to the global config.

```
PROJECT=myapp
MODE=human
```

Commit shared defaults; gitignore personal overrides.

## HTTP API

```bash
fizza serve --addr 127.0.0.1:8080
```

REST surface over the same service layer (local use; no auth).

## Development

Requires Go 1.26+.

```bash
git clone https://github.com/fizza/fizza
cd fizza
make build      # bin/fizza (gitignored)
make test
make test-race
make mcp-test
make install    # $(go env GOPATH)/bin
make fmt
make vet
```

Cross-compile: `make build-all` → `bin/fizza-<os>-<arch>`.

## License

[MIT](LICENSE)
