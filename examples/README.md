# Examples

Copy-paste configs and patterns. Start with [Getting started](../docs/GETTING-STARTED.md) if this is your first install.

## MCP configs (Cursor / Claude Desktop)

| File | Use when |
|------|----------|
| [cursor/mcp.json](cursor/mcp.json) | Default — keyword search, zero extra env |
| [cursor/hybrid-mcp.json](cursor/hybrid-mcp.json) | You want better recall when wording doesn't match exactly (`MEMEX_HYBRID=1`) |
| [mcp-config.json](mcp-config.json) | Minimal snippet (same as cursor default) |

Merge the `mcpServers` block into your client's MCP settings file.

## Guides

| Guide | Topic |
|-------|-------|
| [scoping/README.md](scoping/README.md) | One user, multiple agents, session tags, metadata filters |

## MCP prompts (built into the server)

Available when memex is connected — no extra config:

| Prompt | When |
|--------|------|
| `memory_guide` | Agent needs rules for save vs search vs skip |
| `session_start` | Beginning of a coding session |
| `remember_fact` | Turn rough notes into a clean `remember` call |

Human-readable version: [docs/FOR-AGENTS.md](../docs/FOR-AGENTS.md)

## Verify install

```bash
memex doctor
```
