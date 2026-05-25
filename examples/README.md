# memex examples

Copy-paste recipes for wiring memex into agents and MCP clients.

## Layout

| Path | Purpose |
|------|---------|
| [cursor/mcp.json](cursor/mcp.json) | Minimal Cursor MCP config (stdio) |
| [cursor/hybrid-mcp.json](cursor/hybrid-mcp.json) | Same with `MEMEX_HYBRID=1` for local semantic fusion |
| [scoping/README.md](scoping/README.md) | Multi-agent `user_id` / `agent_id` / `run_id` patterns |

## MCP prompts (v0.6+)

Clients that support MCP prompts can fetch built-in guidance from the running server:

| Prompt | When to use |
|--------|-------------|
| `memory_guide` | Conventions for remember vs recall vs retrieve_context; types, scoping, anti-patterns |
| `session_start` | Start-of-session checklist; optional `run_id` argument |
| `remember_fact` | Distill draft text before calling `remember` (requires `draft`) |

In Cursor, prompts appear when the memex MCP server is connected — use them at session start or when unsure whether to persist a fact.

## CLI doctor

Verify the local store before debugging agent memory issues:

```bash
memex doctor
```

Reports version, schema generation, database path, active memory counts, hybrid mode, and effective env defaults.

## Quick start

```bash
go install github.com/kioie/memex/cmd/memex@latest
memex doctor
```

Then add [cursor/mcp.json](cursor/mcp.json) to your Cursor MCP settings (merge into `mcpServers`).
