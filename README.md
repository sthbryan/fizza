# fizza

A local kanban board for humans and coding agents. One static binary, one SQLite database, three surfaces against the same data.

- **Web UI** — drag-and-drop boards, projects, progress stats
- **CLI** — scriptable, JSON by default, TOON for compact output
- **MCP** — stdio server for Claude Code, Cursor, and similar tools

Data lives in `~/.config/fizza/` (XDG config path on Linux). The binary is CGO-free and statically linked. No accounts, no cloud, no runtime deps.

## Install

**macOS / Linux:**

```bash
curl -fsSL https://raw.githubusercontent.com/sthbryan/fizza/main/scripts/install.sh | sh
```

Picks the right asset for your OS and arch, verifies the SHA256, installs to `/usr/local/bin` (or `~/.local/bin` if that's not writable), and prints the version.

**From source:**

```bash
git clone https://github.com/sthbryan/fizza
cd fizza
make install    # web + build + copy to /usr/local/bin (sudo if needed)
```

Override the install location:

```bash
make install PREFIX=$HOME/.local   # user-local, no sudo
```

Requires Go 1.26+ and Bun.

**Manual download:** pre-built binaries, `.deb`, `.rpm`, and `.apk` packages are on the [releases page](https://github.com/sthbryan/fizza/releases/latest). Archives cover `linux/{amd64,arm64}`, `darwin/{amd64,arm64}`, and `windows/{amd64,arm64}`.

## Quick start

```bash
fizza project new myapp
fizza config set project myapp
fizza config set board main

fizza task add "Ship the first cut"
fizza task add "Write docs" --priority high
fizza task list
fizza task move 1 done

fizza serve              # opens http://127.0.0.1:6500
```

Switch output between JSON (default for agent mode) and tables:

```bash
fizza config set mode human
fizza task list --format pretty
```

## Data model

```
Project
  └── Board (e.g. main)
        └── Columns (todo → in_progress → in_review → done)
              └── Tasks (priority, due date, tags, subtasks)
```

**Task lifecycle:**

| State | How |
|-------|-----|
| Open | in any non-terminal column |
| Done | moved to `done`, `completed`, or `closed`; `completed_at` is set |
| Archived | soft-hidden from the active board (`archived_at`); recoverable |

Listings omit archived tasks and trim done task bodies to counts. Include them with `--include-done` and `--archived`. Archives have their own view in the UI and dedicated CLI commands.

## CLI

| Command | Manages |
|---------|---------|
| `project` | projects (create, list, update, delete) |
| `board` | boards, columns, WIP limits |
| `task` | tasks (add, list, move, update, archive, delete) |
| `tag` | labels |
| `config` | defaults: project, board, output mode |
| `serve` | web UI |
| `mcp` | MCP server over stdio |
| `doctor` | self-checks against the local install |
| `schema` | migrations and integrity |

Common flows:

```bash
fizza task list                  # active work, done omitted
fizza task list --include-done   # include completed
fizza task list --archived       # only archived
fizza task move 12 done
fizza task archive 12
fizza task archive-done          # archive every done task on the current board
fizza task unarchive 12
```

Output formats: `json` (default), `toon` (compact), `pretty` (tables). Override per-command with `--format` or globally with `fizza config set mode human|llm`.

## Web UI

`fizza serve` starts a local HTTP server on `127.0.0.1:6500` (override with `--host`, `--port`, `--addr`) and opens the browser unless you pass `--no-open`. No authentication, so don't expose it beyond localhost.

URLs:

- `/projects`
- `/p/:project/b/:board`
- `/p/:project/b/:board/archived`
- `/stats`

The frontend source is in `web/` (Svelte 5, Vite, TanStack Query, Tailwind). Production assets are embedded into the binary, so `fizza serve` works from any directory.

## Configuration

Global config at `~/.config/fizza/config.json`:

| Key | Effect |
|-----|--------|
| `mode` | `llm` (JSON, default) or `human` |
| `project` | default project for board/task commands |
| `board` | default board when the project has more than one |

```bash
fizza config show
fizza config set project myapp
fizza config set board main
fizza config path
```

Local overrides via a `.fizza` file in the working directory or any parent (walks up to `.git` or `$HOME`):

```
PROJECT=myapp
MODE=human
```

The DB lives next to the config (`default.db`).

## MCP

```bash
fizza mcp
```

Point your client at that stdio command. Agents get the same model and lifecycle rules as the CLI: lean snapshots by default, archive for cleanup. If MCP is not available, shell out to `fizza` and parse JSON from stdout.

Example:

```json
{
  "fizza": {
      "command": "fizza",
      "args": ["mcp"]
   } 
}
```

## Development

Requires Go 1.26+ and Bun.

```bash
make build-full   # web assets + bin/fizza (embedded UI)
make build        # Go-only build (fallback page if web not built)
make web          # bun install + vite build into internal/httpapi/static
make test         # go test ./...
make test-race    # go test -race ./...
make vet          # go vet ./...
make fmt          # gofmt -w + verify clean
make install      # web + build + copy to $(PREFIX)/bin (default /usr/local/bin)
make clean        # remove bin/ + test cache
```

For hot reload while `fizza serve` is running:

```bash
cd web && bun run dev
```

## Design notes

- **SQLite + WAL** — concurrent readers, single writer, durable across crashes.
- **Fractional positions** — inserting a card between two others doesn't reorder the board.
- **Events stream** — local change log; the web UI subscribes via SSE for live updates.
- **WIP limits** — optional caps per column, enforced on move.
- **Lean defaults** — listings omit archived and compress done; opt-in for full payloads.

## License

[MIT](LICENSE) · © 2026 Bryan Villafuerte