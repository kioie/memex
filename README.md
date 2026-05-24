# memex

**Memory extender for AI agents** — local-first, MCP-native, zero config.

Inspired by [Vannevar Bush's memex](https://en.wikipedia.org/wiki/Memex): a device for storing and linking knowledge. This project gives coding agents and LLM clients persistent memory across sessions — no API keys, no cloud, no vector DB setup.

[![Unit Tests](https://github.com/kioie/memex/actions/workflows/unit.yml/badge.svg)](https://github.com/kioie/memex/actions/workflows/unit.yml)
[![Integration Tests](https://github.com/kioie/memex/actions/workflows/integration.yml/badge.svg)](https://github.com/kioie/memex/actions/workflows/integration.yml)
[![SonarQube](https://github.com/kioie/memex/actions/workflows/sonar.yml/badge.svg)](https://github.com/kioie/memex/actions/workflows/sonar.yml)
[![CodeQL](https://github.com/kioie/memex/actions/workflows/codeql.yml/badge.svg)](https://github.com/kioie/memex/actions/workflows/codeql.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/kioie/memex/memex.svg)](https://pkg.go.dev/github.com/kioie/memex/memex)

**Requirements:** Go 1.26+

---

## Quick start

```bash
go install github.com/kioie/memex/cmd/memex@latest
```

Add to Cursor, Claude Desktop, or any MCP client:

```json
{
  "mcpServers": {
    "memex": {
      "command": "memex",
      "args": ["serve"]
    }
  }
}
```

Then tell your agent:

> Remember that I prefer table-driven tests and Go over Python.

Start a new chat later:

> What are my testing preferences?

---

## MCP tools

| Tool | Description |
|------|-------------|
| `remember` | Store a fact (dedups by user_id + content hash; optional metadata) |
| `recall` | FTS search or list recent (filters: tags, type, user_id, pagination) |
| `list_memories` | Filtered list without search query (mem0 `get_memories`) |
| `update_memory` | Overwrite content by ID |
| `get_memory` | Fetch one memory by exact ID |
| `forget` | Delete one memory by ID |
| `delete_memories` | Batch delete by IDs |
| `delete_all_memories` | Wipe user scope (requires `confirm=true`) |
| `memory_history` | Audit trail for a memory |

Set `MEMEX_USER_ID` to scope memories (mem0-style `user_id`, default `default`).

Memories live in `~/.memex/memex.db` (override with `MEMEX_DIR`).

---

## Why memex?

| | memex | mem0 / hosted memory |
|---|---|---|
| **Setup** | `go install`, one MCP config line | API keys, often Docker |
| **Data** | Local SQLite on your machine | Vendor cloud |
| **Network** | Zero for recall | Every search hits an API |
| **MCP-native** | Built as an MCP server | SDK wrapper or separate API |

---

## Development

```bash
git clone https://github.com/kioie/memex.git
cd memex
make test              # fast unit suite (race + -short)
make test-integration  # MCP stdio subprocess roundtrip
make test-full         # scale tests (1k–5k rows, large payloads)
go run ./cmd/memex serve
```

Layer split: `memex/store.go` owns SQLite + FTS; `memex/server.go` registers tools via `tinymcp.RegisterTool` and serves over stdio (`server.Start`). Roundtrip tests use `server.RawServer()` for in-memory transport; integration tests spawn the CLI subprocess.

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full test tier breakdown and [SECURITY.md](SECURITY.md) for reporting vulnerabilities.

Set `MEMEX_VERBOSE=1` to log the database path to stderr.

---

## License

MIT — see [LICENSE](LICENSE).
