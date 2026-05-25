# Multi-agent scoping

memex partitions memory with `user_id`, optional `agent_id`, and optional `run_id`. Use env defaults in MCP config so tools inherit scope without repeating arguments.

## Single user, single agent (default)

```json
{
  "mcpServers": {
    "memex": {
      "command": "memex",
      "args": ["serve"],
      "env": {
        "MEMEX_USER_ID": "eddy"
      }
    }
  }
}
```

All remembers and recalls default to `user_id=eddy`.

## Multiple agents, one user

Run separate MCP server entries or swap `MEMEX_AGENT_ID` per agent profile:

```json
{
  "mcpServers": {
    "memex-reviewer": {
      "command": "memex",
      "args": ["serve"],
      "env": {
        "MEMEX_USER_ID": "eddy",
        "MEMEX_AGENT_ID": "code-reviewer"
      }
    },
    "memex-implementer": {
      "command": "memex",
      "args": ["serve"],
      "env": {
        "MEMEX_USER_ID": "eddy",
        "MEMEX_AGENT_ID": "implementer"
      }
    }
  }
}
```

Recall with explicit filters when sharing a store:

```json
{
  "query": "testing preferences",
  "user_id": "eddy",
  "agent_id": "implementer"
}
```

## Session / run tagging

Tag memories from one chat or CI run without a separate database:

```json
{
  "mcpServers": {
    "memex": {
      "command": "memex",
      "args": ["serve"],
      "env": {
        "MEMEX_USER_ID": "eddy",
        "MEMEX_RUN_ID": "cursor-session-2026-05-24"
      }
    }
  }
}
```

Or pass `run_id` on individual `remember` calls. Filter later with `run_id` on `recall`, `list_memories`, or `retrieve_context`.

## Metadata filters

Store structured tags for project or repo scoping:

```json
{
  "content": "API base URL is https://api.example.com/v2",
  "type": "fact",
  "tags": ["api", "backend"],
  "metadata": {
    "project": "payments",
    "repo": "github.com/acme/payments"
  }
}
```

Recall with exact metadata equality:

```json
{
  "query": "API base",
  "metadata": {
    "project": "payments"
  }
}
```

## Agent vs user facts

| Scenario | type | source |
|----------|------|--------|
| User says "I prefer Go" | preference | user (default) |
| Agent says "I'll add tests next" | commitment | agent |
| Agent recommends a library | recommendation | agent |
| Agent merged a PR | action_taken | agent |

Filter agent-only context: `"source": "agent"` on recall or `retrieve_context`.

## Isolation checklist

- Never share `user_id` across tenants — it is the hard boundary.
- Use `agent_id` when multiple agents write to the same user store.
- Use `run_id` or metadata for ephemeral session groupings, not secrets.
- Run `memex doctor` to confirm which defaults the CLI process sees.
