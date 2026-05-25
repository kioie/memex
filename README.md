# memex

**Memory extender for AI agents** — local-first, MCP-native, zero config. **v0.5.1**

Inspired by [Vannevar Bush's memex](https://en.wikipedia.org/wiki/Memex): a device for storing and linking knowledge. This project gives coding agents and LLM clients durable memory across sessions — no API keys, no cloud, no vector DB setup.

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

For bounded context injection, use `retrieve_context` with a token budget instead of dumping full recall results.

---

## What you get (v0.5.0)

| Capability | memex |
|------------|-------|
| **Hybrid retrieval** | FTS5 + BM25, entity boost, optional local vectors — fused with reciprocal rank fusion |
| **Token budget** | `retrieve_context` packs ranked hits into a configurable `max_tokens` ceiling |
| **Append-only facts** | `update_memory` supersedes (new row); soft-delete preserves audit trail |
| **Agent facts** | First-class `source` (`user` / `agent` / `system`) and commitment-style types |
| **Strong scoping** | `user_id`, `agent_id`, `run_id`, metadata filters — scoped CRUD and history |
| **Fully local** | SQLite + FTS5 on disk; optional semantic signal via `MEMEX_HYBRID=1` — no cloud LLM or embeddings API |

See [CHANGELOG.md](CHANGELOG.md) and [docs/ROADMAP-v0.3.md](docs/ROADMAP-v0.3.md) for release history and future work.

---

## MCP tools

| Tool | Description |
|------|-------------|
| `remember` | Store a fact (hash dedup per user; optional tags, type, source, metadata, agent/run scope) |
| `recall` | Keyword search (FTS5 + hybrid fusion); **query required** — use `list_memories` to browse |
| `retrieve_context` | Ranked search packed within `max_tokens` (greedy; hybrid fusion before packing) |
| `list_memories` | Filtered, paginated browse without a search query |
| `update_memory` | Supersede an active memory (closes old row, returns new ID) |
| `get_memory` | Fetch one memory by exact ID (scoped) |
| `forget` | Soft-delete one memory (`valid_to` set; row kept for audit) |
| `delete_memories` | Batch soft-delete by IDs |
| `delete_all_memories` | Wipe user scope (requires `confirm=true`) |
| `memory_history` | ADD / supersede / delete audit trail for a memory |

Memory types include `note`, `preference`, `decision`, `fact`, `procedure`, plus agent-oriented types: `commitment`, `recommendation`, `action_taken`.

---

## Environment

| Variable | Purpose |
|----------|---------|
| `MEMEX_DIR` | Data directory (default `~/.memex`, database at `memex.db`) |
| `MEMEX_USER_ID` | Default user scope (default `default`) |
| `MEMEX_AGENT_ID` | Default agent scope when tool args omit `agent_id` |
| `MEMEX_RUN_ID` | Default run/session tag when tool args omit `run_id` |
| `MEMEX_HYBRID=1` | Enable local vector retrieval (deterministic embeddings, fused with keyword + entity signals) |
| `MEMEX_VERBOSE=1` | Log database path to stderr (stdio reserved for MCP) |

---

## Why memex?

Most agent memory products assume a hosted stack: API keys, cloud embeddings, and network round-trips on every recall. memex inverts that — the MCP server *is* the product, and your machine owns the data.

| | memex | Typical hosted memory |
|---|---|---|
| **Setup** | `go install`, one MCP config line | API keys, SDK, often Docker |
| **Data residency** | Local SQLite on your machine | Vendor cloud |
| **Recall path** | On-disk FTS + optional local vectors | Remote API + embedding service |
| **Token discipline** | Built-in `retrieve_context` budget | Often returns unbounded JSON |
| **Fact lifecycle** | Append-only supersession + soft-delete audit | In-place UPDATE/DELETE |
| **Agent scoping** | `user_id` / `agent_id` / `run_id` / `source` | Varies; often single-tenant |
| **MCP-native** | First-class tools with clear browse vs search split | Wrapper around REST or proprietary SDK |
| **Dependencies** | Go + SQLite (stdlib-style local stack) | Vector DB, LLM pipeline, or both |

memex deliberately does **not** run an LLM extraction pipeline or call cloud embedding APIs — agents write distilled facts directly, which keeps latency predictable and avoids vendor lock-in.

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

---

## License

MIT — see [LICENSE](LICENSE).
