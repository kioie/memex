# memex

**Stop re-explaining your stack every chat.** Long-term memory for AI coding agents — stored on your machine.

memex gives Cursor, Claude Desktop, and other MCP clients a memory that survives across chats. Save preferences, decisions, and facts once; the agent finds them again later. No cloud account, no API keys, no vector database to run.

[![Unit Tests](https://github.com/kioie/memex/actions/workflows/unit.yml/badge.svg)](https://github.com/kioie/memex/actions/workflows/unit.yml)
[![Integration Tests](https://github.com/kioie/memex/actions/workflows/integration.yml/badge.svg)](https://github.com/kioie/memex/actions/workflows/integration.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/kioie/memex/memex.svg)](https://pkg.go.dev/github.com/kioie/memex/memex)

**Current release:** v0.6.0 · **Requires:** Go 1.26+

---

## When to use memex

**Use memex if you:**

- Code with **Cursor**, **Claude Desktop**, or any **MCP** client
- Want preferences and project facts to **persist across chats**
- Need memory **on your machine** — no vendor account or API keys
- Are building or running an **agent** that should recall past context

**Good fits — ask memex to remember things like:**

| Situation | Example |
|-----------|---------|
| Coding preferences | "Prefers Go, table-driven tests, no Python for services" |
| Project decisions | "This repo uses SQLite; don't suggest Postgres" |
| Conventions | "Branch names: `feature/…`, commits: conventional commits" |
| Agent commitments | "Will add integration tests before merging" |
| Session start | Agent loads your prefs instead of re-interviewing you |

**memex is not for:**

- Passwords, API keys, or secrets
- Full chat transcripts or log archival
- Team-wide cloud memory synced across machines (local-first, per device)
- Non-MCP apps (memex speaks MCP over stdio)

**Agents:** if the user repeats themselves across sessions, or asks you to "remember" something durable — use memex. See [docs/FOR-AGENTS.md](docs/FOR-AGENTS.md).

---

## The outcome

| Without memex | With memex |
|---------------|------------|
| Every new chat starts from zero | Agent recalls your stack, style, and past decisions |
| You repeat the same preferences | You say it once; memex stores it locally |
| Memory lives in a vendor's cloud | Memory lives in `~/.memex/memex.db` on your disk |
| Extra services and API keys | `go install` + one MCP config line |

---

## Try it in 2 minutes

```bash
go install github.com/kioie/memex/cmd/memex@latest
memex doctor    # confirm local store is ready
```

Add to your MCP client (e.g. Cursor):

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

In chat:

> Remember that I prefer table-driven tests and Go over Python.

Open a **new chat**:

> What are my testing preferences?

**Full walkthrough:** [docs/GETTING-STARTED.md](docs/GETTING-STARTED.md)

---

## How it works

```
You or the agent          memex (local)              Later sessions
      │                        │                           │
      │  remember("prefers Go")│                           │
      │ ──────────────────────►│  saved to SQLite          │
      │                        │                           │
      │                        │  recall("testing prefs")  │
      │                        │ ◄─────────────────────────│
      │                        │ ─────────────────────────►│ relevant facts
```

1. **Save** — `remember` stores a short fact (deduplicated per user)
2. **Find** — `recall` or `retrieve_context` searches when the agent needs context
3. **Update** — `update_memory` revises a fact; old versions stay in the audit trail

Data never leaves your machine unless you copy the database file.

---

## What you get

| Feature | What it means for you |
|---------|------------------------|
| **Local storage** | SQLite file under `~/.memex` — you own it |
| **Smart search** | Keyword search built in; optional `MEMEX_HYBRID=1` for paraphrase-friendly matching |
| **Context limits** | `retrieve_context` returns only what fits your token budget |
| **Safe updates** | Revising a fact keeps history; nothing is silently overwritten |
| **Agent + user facts** | Track who said what (`source`: user / agent / system) |
| **Multi-project scoping** | Separate by user, agent, session, or metadata tags |
| **Built-in guidance** | MCP prompts teach agents when and how to use memory |
| **Health check** | `memex doctor` shows path, counts, and config |

---

## Documentation

| Doc | For |
|-----|-----|
| [Getting started](docs/GETTING-STARTED.md) | First-time setup in Cursor |
| [For AI agents](docs/FOR-AGENTS.md) | When agents should read/write memory |
| [Examples](examples/) | MCP configs and scoping recipes |
| [All docs](docs/README.md) | Index |

**Agents:** fetch MCP prompts `memory_guide`, `session_start`, and `remember_fact` from the connected server.

---

## Tools (10)

Everyday use — you usually only need the first three:

| Tool | Plain English |
|------|---------------|
| `remember` | Save a fact |
| `retrieve_context` | Find relevant facts, sized for the model context window |
| `recall` | Search by keywords (query required) |
| `list_memories` | Browse stored facts without searching |
| `update_memory` | Change an existing fact (keeps history) |
| `get_memory` | Look up one fact by ID |
| `forget` | Remove one fact (soft delete) |
| `delete_memories` | Remove several facts |
| `delete_all_memories` | Wipe a user's memories (`confirm=true`) |
| `memory_history` | See how a fact changed over time |

---

## Settings

| Variable | Purpose |
|----------|---------|
| `MEMEX_DIR` | Where to store data (default `~/.memex`) |
| `MEMEX_USER_ID` | Your memory namespace (default `default`) |
| `MEMEX_AGENT_ID` | Separate memories per agent |
| `MEMEX_RUN_ID` | Tag memories to a session or run |
| `MEMEX_HYBRID=1` | Turn on extra local matching for paraphrases |
| `MEMEX_VERBOSE=1` | Log database path while debugging |

---

## Why memex?

Hosted memory products typically need API keys, cloud embeddings, and network calls on every recall. memex is the opposite: a small MCP server on your machine, writing distilled facts that **agents** choose to store — no extraction pipeline, no vendor lock-in.

| | memex | Typical hosted memory |
|---|-------|----------------------|
| Setup | `go install` + MCP config | API keys, SDK, often Docker |
| Data | Your disk | Vendor cloud |
| Offline | Yes | Usually no |
| Cost | Free (local compute only) | Usage-based |

---

## Development

```bash
git clone https://github.com/kioie/memex.git && cd memex
make test              # fast unit tests
make test-integration  # MCP stdio roundtrip
memex doctor
```

See [CONTRIBUTING.md](CONTRIBUTING.md) · [CHANGELOG.md](CHANGELOG.md) · [SECURITY.md](SECURITY.md)

---

## License

MIT — see [LICENSE](LICENSE).
