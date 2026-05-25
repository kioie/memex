# Smithery publishing

List **memex** on [Smithery](https://smithery.ai) with **MCPB (stdio Docker)** — users install via Smithery and run the published image locally. **No HTTP hosting required.**

| Mode | Who runs what? | Hosting cost |
|------|----------------|--------------|
| **MCPB (stdio Docker)** ✓ recommended | User runs Docker via Smithery install | **$0** for you |
| **URL (streamable HTTP)** | You host HTTPS; Smithery proxies | PaaS bill (optional) |

**Privacy note:** For personal, durable memory use **local stdio** (`memex serve` in Cursor) or Smithery MCPB. Data stays on the user's machine in a Docker volume.

**Live listing:** https://smithery.ai/servers/kioie/memex

---

## MCPB listing (stdio / local Docker)

Smithery's install flow runs `ghcr.io/kioie/memex:0.6.0` with a local `memex-data` volume at `/data`.

### Prerequisites

- Docker image published: `ghcr.io/kioie/memex:0.6.0` (see root [`Dockerfile`](../Dockerfile) and CI workflow)
- [`smithery.yaml`](../smithery.yaml) and [`smithery/entry.js`](../smithery/entry.js)

### Build and publish

```bash
smithery auth login
npx mcp-bundler bundle --entry=smithery/entry.js --inspect
npx @anthropic-ai/mcpb pack .smithery/mcpb server.mcpb
smithery mcp publish server.mcpb -n kioie/memex
```

Users connect:

```bash
npx -y smithery mcp add kioie/memex
```

---

## HTTP listing (optional — self-host only)

Only if you want a remote try-it URL without Docker on the client. Requires you to host HTTPS (Render, Fly, Koyeb, etc.).

Template: [`examples/http-deploy/`](../examples/http-deploy/) — not needed for the default Smithery MCPB listing.

```bash
smithery mcp publish "https://YOUR_HOST" -n kioie/memex
```

See [`examples/http-deploy/README.md`](../examples/http-deploy/README.md) for endpoints and platform configs.

---

## After publishing

- Link from [README](../README.md) and [DISCOVERY.md](./DISCOVERY.md)
- Tag releases align image tags with git tags (e.g. `v0.6.0`)
- Rebuild and republish MCPB when tools or `smithery.yaml` change

See also: [MCP Registry](./DISCOVERY.md#official-mcp-registry) via [`server.json`](../server.json).
