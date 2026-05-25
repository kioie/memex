# Getting started

This guide gets memex running in Cursor (or any MCP client) in a few minutes.

## What you'll have when done

- A local database at `~/.memex/memex.db` holding your agent's memories
- An MCP server your agent can call to **save**, **find**, and **update** facts
- A way to confirm everything works (`memex doctor`)

No signup, no API keys, no Docker.

---

## 1. Install

Requires [Go 1.26+](https://go.dev/dl/).

```bash
go install github.com/kioie/memex/cmd/memex@latest
```

Confirm:

```bash
memex version
# 0.6.0
```

---

## 2. Check the local store

```bash
memex doctor
```

You should see something like:

```
memex doctor (0.6.0)
schema:      5
database:    /Users/you/.memex/memex.db
active:      0 memories (0 in user_id="default")
hybrid:      disabled (0 embeddings indexed)
...
status:      ok
```

If `status: ok`, the database is ready.

---

## 3. Add to Cursor

Open **Cursor Settings → MCP** and add a server. Minimal config:

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

Copy-paste ready files: [examples/cursor/mcp.json](../examples/cursor/mcp.json)

Restart Cursor or reload MCP so the server connects.

---

## 4. Try it in chat

**Save a preference:**

> Remember that I prefer table-driven tests and Go over Python.

**New chat later:**

> What are my testing preferences?

The agent should call memex to recall what you saved.

---

## 5. Optional settings

| Variable | When to set it |
|----------|----------------|
| `MEMEX_USER_ID` | Separate memories per person or project (default: `default`) |
| `MEMEX_HYBRID=1` | Better matching when wording differs from what was saved ([hybrid config](../examples/cursor/hybrid-mcp.json)) |
| `MEMEX_DIR` | Custom folder instead of `~/.memex` |
| `MEMEX_VERBOSE=1` | Log database path to stderr while debugging |

Example with a personal user id:

```json
{
  "mcpServers": {
    "memex": {
      "command": "memex",
      "args": ["serve"],
      "env": {
        "MEMEX_USER_ID": "your-name"
      }
    }
  }
}
```

---

## Troubleshooting

| Problem | Try |
|---------|-----|
| Agent doesn't use memex | Confirm MCP shows memex connected; ask explicitly: "Use memex to remember this" |
| `command not found: memex` | Ensure `$GOPATH/bin` or `$(go env GOPATH)/bin` is on your `PATH` |
| Wrong or empty memories | Run `memex doctor` — check `user_id` matches your MCP env |
| Need multi-agent isolation | See [examples/scoping](../examples/scoping/README.md) |

---

## Next steps

- [For AI agents](FOR-AGENTS.md) — rules for when agents should read/write memory
- [Examples](../examples/) — hybrid mode, scoping, prompts
- [README](../README.md) — full feature overview
