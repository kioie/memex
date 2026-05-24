# memex

**Memory extender for AI agents** — local-first, MCP-native, zero config.

Inspired by [Vannevar Bush's memex](https://en.wikipedia.org/wiki/Memex): a device for storing and linking knowledge. This project gives coding agents and LLM clients persistent memory across sessions — no API keys, no cloud, no vector DB setup.

[![Unit Tests](https://github.com/kioie/memex/actions/workflows/unit.yml/badge.svg)](https://github.com/kioie/memex/actions/workflows/unit.yml)
[![Integration Tests](https://github.com/kioie/memex/actions/workflows/integration.yml/badge.svg)](https://github.com/kioie/memex/actions/workflows/integration.yml)
[![CodeQL](https://github.com/kioie/memex/actions/workflows/codeql.yml/badge.svg)](https://github.com/kioie/memex/actions/workflows/codeql.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/kioie/memex/memex.svg)](https://pkg.go.dev/github.com/kioie/memex/memex)

Built with [tiny-go-mcp-server](https://github.com/kioie/tiny-go-mcp-server) and the official [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk).

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
| `remember` | Store a fact, preference, decision, or note |
| `recall` | Search memories (FTS5) or list recent entries |
| `forget` | Delete a memory by ID |
| `get_memory` | Fetch one memory by exact ID |

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

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full test tier breakdown and [SECURITY.md](SECURITY.md) for reporting vulnerabilities.

Set `MEMEX_VERBOSE=1` to log the database path to stderr.

---

## License

MIT — see [LICENSE](LICENSE).
