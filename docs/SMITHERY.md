# Smithery publishing

Two ways to list **memex** on [Smithery](https://smithery.ai):

| Mode | Who runs what? | Best for |
|------|----------------|----------|
| **URL (streamable HTTP)** | You host HTTPS; users connect via Smithery Gateway | Try-it demos, remote MCP without local install |
| **MCPB (stdio bundle)** | User runs Docker via Smithery install | Local memory with Smithery-managed config |

**Privacy note:** For personal, durable memory use **local stdio** (`memex serve` in Cursor). Hosted HTTP is for demos or **self-hosted** deployments you control.

## URL listing (remote try-it)

Smithery proxies to your **public HTTPS** streamable HTTP endpoint.

### 1. Deploy the HTTP example

Template: [`examples/http-deploy/`](../examples/http-deploy/).

```bash
mkdir -p ./data
MEMEX_DIR=./data go run ./examples/http-deploy
curl -s http://127.0.0.1:8080/health
```

Production options:

- **Render:** [`render.yaml`](../examples/http-deploy/render.yaml) — root dir `examples/http-deploy`
- **Railway:** deploy from `examples/http-deploy` (sets `PORT`)
- **Fly.io:** [`fly.toml`](../examples/http-deploy/fly.toml) + optional volume at `/data`

Endpoints:

| Path | Purpose |
|------|---------|
| `/` | Streamable HTTP MCP |
| `/health` | Health check |
| `/.well-known/mcp/server-card.json` | Metadata for Smithery scan |

Set **`MEMEX_DIR=/data`** in containers. Mount persistent storage if memories should survive redeploys.

### 2. Publish on Smithery

```bash
smithery auth login
smithery mcp publish "https://YOUR_HOST" -n kioie/memex
```

Requirements:

- Public **HTTPS** URL (no trailing slash)
- Streamable HTTP at `/` (this repo uses `tinymcp.StreamableHTTPHandler`)
- Return **401** (not 403) if you add auth later

Local tunnel for testing:

```bash
ngrok http 8080
smithery mcp publish "https://YOUR_SUBDOMAIN.ngrok-free.app" -n kioie/memex
```

### 3. Server card

If WAF blocks Smithery's scan, metadata is served at:

`/.well-known/mcp/server-card.json`

See [`examples/http-deploy/server-card.json`](../examples/http-deploy/server-card.json).

---

## MCPB listing (stdio / local Docker)

For Smithery's install flow running memex in Docker with a local volume:

1. Build and push the stdio image (see root [`Dockerfile`](../Dockerfile)):

```bash
docker build -t ghcr.io/kioie/memex:0.6.0 .
docker push ghcr.io/kioie/memex:0.6.0
```

2. Config: [`smithery.yaml`](../smithery.yaml) — mounts `memex-data` → `/data`

3. Publish MCPB per [Smithery MCPB docs](https://smithery.ai/docs/build/publish).

---

## After publishing

- Link from [README](../README.md) and [DISCOVERY.md](./DISCOVERY.md)
- Update GitHub About homepage to Smithery URL when live
- Monitor: hosted demos may need rate limits or auth if traffic grows

See also: [MCP Registry](./DISCOVERY.md#official-mcp-registry) via [`server.json`](../server.json).
