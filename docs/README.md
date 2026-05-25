# memex documentation

Plain-language guides for humans and AI agents.

| Guide | Audience | What you'll learn |
|-------|----------|-------------------|
| [Getting started](GETTING-STARTED.md) | Developers | Install, connect Cursor, verify it works, first memories |
| [For AI agents](FOR-AGENTS.md) | LLMs / agent authors | Which tool to call, when to save memory, what to avoid |
| [Examples](../examples/) | Everyone | Copy-paste MCP configs and multi-agent setups |
| [Discovery](DISCOVERY.md) | Maintainers | MCP Registry, Smithery, directories |
| [Branding](BRANDING.md) | Maintainers | Logo, banner, and listing assets |
| [Smithery](SMITHERY.md) | Maintainers | Publish MCPB bundle (stdio Docker) |
| [Roadmap](ROADMAP-v0.3.md) | Contributors | Release history and planned work |
| [Changelog](../CHANGELOG.md) | Everyone | Version-by-version changes |

## One-sentence summary

**memex** is a local memory server for MCP clients: agents save short facts once and find them again in future chats — no cloud account required.

## Quick links

- **Install:** `go install github.com/kioie/memex/cmd/memex@latest`
- **Health check:** `memex doctor`
- **Run server:** `memex serve` (stdio — used by MCP clients automatically)
- **Try in browser:** `npx @modelcontextprotocol/inspector memex serve`
- **HTTP deploy (optional):** [examples/http-deploy](../examples/http-deploy/)
- **Repo:** [github.com/kioie/memex](https://github.com/kioie/memex)
