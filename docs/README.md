# memex documentation

Plain-language guides for humans and AI agents.

| Guide | Audience | What you'll learn |
|-------|----------|-------------------|
| [Getting started](GETTING-STARTED.md) | Developers | Install, connect Cursor, verify it works, first memories |
| [For AI agents](FOR-AGENTS.md) | LLMs / agent authors | Which tool to call, when to save memory, what to avoid |
| [Examples](../examples/) | Everyone | Copy-paste MCP configs and multi-agent setups |
| [Roadmap](ROADMAP-v0.3.md) | Contributors | Release history and planned work |
| [Changelog](../CHANGELOG.md) | Everyone | Version-by-version changes |

## One-sentence summary

**memex** is a local memory server for MCP clients: agents save short facts once and find them again in future chats — no cloud account required.

## Quick links

- **Install:** `go install github.com/kioie/memex/cmd/memex@latest`
- **Health check:** `memex doctor`
- **Run server:** `memex serve` (stdio — used by MCP clients automatically)
- **Repo:** [github.com/kioie/memex](https://github.com/kioie/memex)
