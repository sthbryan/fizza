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
| `tag_add` | Create a tag in a project |
| `tag_list` | List tags in a project |
| `tag_delete` | Delete a tag (cascades) |
| `tag_attach` | Attach a tag to a task |
| `tag_detach` | Detach a tag from a task |

All tools accept and return JSON. The MCP server enforces the same schema; invalid inputs are rejected before any DB write.

## Configuration

User config lives at `~/.config/fizza/config.json` (XDG-aware). Three fields:

| Key | Values | Effect |
|---|---|---|
| `mode` | `llm` (default) \| `human` | When `human`, output is rendered as tables without passing `--format` |
| `project` | project name | Default project for board/task commands. Override per-call by changing it |
| `board` | board name | Default board within the project. Most commands (`fizza task add/list/move`) skip `--board` when this is set or the project has exactly one board |

```bash
fizza config show                   # show current config
fizza config set mode human         # human-readable output by default
fizza config set project alpha      # make `alpha` the default project
fizza config set board main         # make `main` the default board within `alpha`
fizza config unset project          # clear the default (board/task commands will require it)
fizza config path                   # print the config file path
```

### Per-project config (`.fizza`)

For mono-repo or multi-project workflows, drop a `.fizza` file at the project root. It uses env-style `KEY=VALUE` lines and overrides the global config when fizza runs inside that directory or any subdirectory (search walks up to 5 levels and stops at the first `.git`, `$HOME`, or filesystem root).

```bash
# .fizza
PROJECT=myapp          # default project for this checkout
MODE=human             # default output mode (llm | human)
```

The closest `.fizza` to your cwd wins; keys not set fall back to the global config. Add `.fizza` to `.gitignore` if it's personal state; commit it if it's shared team config.


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

## License

MIT
