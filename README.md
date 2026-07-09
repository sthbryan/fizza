# fizza

**Local kanban that works the same for you and your coding agents.**

One static binary. One SQLite database. A CLI, a web board, and an [MCP](https://modelcontextprotocol.io/) server that all share the same data—so work you track in the browser is the same work an agent can list, move, and complete without a cloud SaaS in the middle.

---

## Why fizza

Most task tools are either built for humans (pretty UI, awkward for scripts) or for agents (JSON dumps that grow forever and blow the context window). fizza is both:

| Surface | Role |
|---------|------|
| **Web UI** | Drag-and-drop boards, projects, progress stats |
| **CLI** | Scriptable, structured output (JSON by default) |
| **MCP** | Same model over stdio for Claude Code, Cursor, and similar tools |

Data stays on your machine under the XDG config path (`~/.config/fizza/` on Linux/macOS). No accounts, no remote API, no runtime deps—the binary is CGO-free and statically linked.

---

## Model

```
Project
  └── Board (e.g. main)
        └── Columns (todo → in_progress → in_review → done)
              └── Tasks (priority, due date, tags, subtasks)
```

**Task lifecycle**

1. **Open** — work in non-terminal columns  
2. **Done** — moved to `done` / `completed` / `closed` (`completed_at` is set)  
3. **Archived** — soft-hidden from the active board (`archived_at`); recoverable later  

By default, listings and board snapshots keep **done bodies lean** (counts instead of full card dumps) and **omit archived tasks**, so agents and UIs stay focused on active work. Completed work can be shown on demand; archived work has its own view in the UI and dedicated CLI commands.

---

## Build

Requires **Go 1.26+**. For the embedded web UI, also **Bun**.

```bash
git clone <your-repo-url>
cd fizza
make build-full   # builds web assets + bin/fizza
# or, Go only (fallback page if web was never built):
make build
```

```bash
make test
make install      # into $(go env GOPATH)/bin
```

---

## Quick start

```bash
fizza project new myapp
fizza config set project myapp
fizza config set board main

fizza task add "Ship the first cut"
fizza task add "Write docs" --priority high
fizza task list
fizza task move 1 done
```

Open the board in the browser:

```bash
fizza serve
# http://127.0.0.1:6500
```

Prefer tables over JSON:

```bash
fizza config set mode human
# or one-shot:
fizza task list --format pretty
```

---

## Web UI

`fizza serve` starts a local HTTP server (default `127.0.0.1:6500`) and opens the browser unless you pass `--no-open`.

**What you get**

- Project list and boards  
- Kanban with drag-and-drop  
- Show / hide completed columns  
- Archive completed work; dedicated archived view  
- Progress stats (by project, board, column, priority, activity)  

Same database as the CLI and MCP. Navigation is URL-based (`/projects`, `/p/:project/b/:board`, `/stats`). Bindings can be changed with `--host`, `--port`, or `--addr`. There is no authentication—keep it on localhost.

**Frontend source** lives in `web/` (Svelte 5, Vite, TanStack Query, Tailwind). Production assets embed into the binary:

```bash
make web          # bun install + vite build
make build-full   # web + Go binary

# optional hot reload while serve is running:
cd web && bun run dev
```

---

## CLI overview

| Command | Purpose |
|---------|---------|
| `project` | Create, list, update, delete projects |
| `board` | Create boards, columns, WIP limits |
| `task` | Add, list, move, update, archive, delete |
| `tag` | Labels attachable to tasks |
| `config` | Defaults: project, board, output mode |
| `serve` | Web UI |
| `mcp` | MCP server (stdio) |
| `doctor` | Installation self-checks |
| `schema` | Migrations and integrity |

Common task flows:

```bash
fizza task list                      # active work (done omitted by default)
fizza task list --include-done
fizza task list --archived
fizza task move 12 done
fizza task archive 12
fizza task archive-done              # archive all done on the current board
fizza task unarchive 12
```

Output formats: `json` (default for LLM mode), `toon` (compact), `pretty` (tables). Use `--format` or `fizza config set mode human|llm`.

---

## Configuration

**Global:** `~/.config/fizza/config.json`

| Key | Meaning |
|-----|---------|
| `mode` | `llm` (JSON-oriented, default) or `human` |
| `project` | Default project for board/task commands |
| `board` | Default board when a project has more than one |

```bash
fizza config show
fizza config set project myapp
fizza config set board main
fizza config path
```

**Local overrides:** a `.fizza` file in a repo (walks up a few directories; stops at `.git` / home). Closest file wins; missing keys fall back to global config.

```
PROJECT=myapp
MODE=human
```

Database path defaults under the same XDG config directory (`default.db`).

---

## Agents (MCP)

```bash
fizza mcp
```

Point your client at that stdio command (Claude Code, Cursor, Continue, etc.). Agents use the same projects, boards, and lifecycle rules as the CLI: lean snapshots by default, archive for long-term cleanup.

You can also shell out to `fizza` and parse JSON from stdout if MCP is not available.

---

## Design notes

- **SQLite + WAL** — concurrent readers, sensible write behavior for a desktop tool  
- **Fractional positions** — stable ordering when inserting cards between others  
- **Events stream** — local change log; the web UI can live-update via SSE  
- **WIP limits** — optional per-column caps  
- **No cloud** — intentional: your board data never leaves the machine unless you move the DB file yourself  

---

## License

[MIT](LICENSE) · © 2026 Bryan Villafuerte
