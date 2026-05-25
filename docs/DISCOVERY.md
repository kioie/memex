# Discovery and publishing

Make **memex** visible to developers, agents, and MCP directories.

## Positioning

> **Stop re-explaining your stack every chat.** Local MCP memory for Cursor and Claude — facts persist on your machine, no cloud signup.

Use this in directory listings, Smithery, and social posts.

## Official MCP Registry

Metadata: [`server.json`](../server.json) at repo root.

1. Install publisher CLI: https://modelcontextprotocol.io/registry/quickstart
2. Authenticate: `mcp-publisher login github`
3. Validate: `mcp-publisher validate` (from repo root)
4. Publish: `mcp-publisher publish`
5. Tag releases align `server.json` version with git tags (e.g. `v0.6.0`)

Docker package (stdio): `ghcr.io/kioie/memex:<version>` — build with root [`Dockerfile`](../Dockerfile).

Logo and banner: [`docs/BRANDING.md`](./BRANDING.md) · [`docs/assets/`](../docs/assets/)

## Smithery

**Live:** https://smithery.ai/servers/kioie/memex — MCPB stdio Docker (no hosting cost).

| Mode | Doc | Best for |
|------|-----|----------|
| **MCPB (stdio Docker)** ✓ | [SMITHERY.md](./SMITHERY.md) · [`smithery.yaml`](../smithery.yaml) | Smithery install → local Docker volume at `/data` |
| **URL (HTTP)** optional | [`examples/http-deploy/`](../examples/http-deploy/) | Self-hosted remote try-it only |

## Community directories (manual)

| Directory | Notes |
|-----------|--------|
| [awesome-mcp-servers](https://github.com/punkpeye/awesome-mcp-servers) | PR with one-liner + link |
| [mcp.so](https://mcp.so) | Submit server blurb |
| [Glama](https://glama.ai/mcp/servers) | GitHub + Dockerfile |

## GitHub repository

- **About** description and **topics** (`mcp`, `ai-agents`, `cursor`, `memory`, `golang`, …)
- Homepage → [Getting started](./GETTING-STARTED.md)
- Pin releases with `go install …@vX.Y.Z`

## Try without Cursor (local sandbox)

```bash
go install github.com/kioie/memex/cmd/memex@latest
npx @modelcontextprotocol/inspector memex serve
```

Interactive browser UI for all tools and prompts.

## Related docs

- [Getting started](./GETTING-STARTED.md) — local stdio setup
- [For AI agents](./FOR-AGENTS.md) — tool conventions
- [Examples](../examples/) — MCP configs
