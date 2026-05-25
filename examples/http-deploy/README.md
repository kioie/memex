# HTTP deploy example (optional — self-host only)

Host **memex** as streamable HTTP MCP for self-hosted remote access or any HTTP MCP client.

**Smithery listing uses MCPB (stdio Docker) instead** — see [docs/SMITHERY.md](../../docs/SMITHERY.md). You do not need this example for Smithery unless you want a hosted try-it URL.

| Path | Purpose |
|------|---------|
| `/` | Streamable HTTP MCP |
| `/health` | Platform health check |
| `/.well-known/mcp/server-card.json` | Static metadata for Smithery scan |

Full guide: [docs/SMITHERY.md](../../docs/SMITHERY.md).

## Important: hosted vs local

| Mode | Best for |
|------|----------|
| **Local stdio** (`memex serve`) | Private, durable memory on your machine (recommended) |
| **Self-hosted HTTP** (this example) | Your team, with persistent disk and auth at the edge |
| **Public demo HTTP** | Trying memex in Smithery — data may reset on redeploy |

For production personal memory, use local stdio. Use HTTP when you control hosting, storage, and access.

## Run locally

```bash
mkdir -p ./data
MEMEX_DIR=./data go run ./examples/http-deploy
# MCP: http://127.0.0.1:8080/
curl -s http://127.0.0.1:8080/health
```

Optional tunnel for Smithery testing:

```bash
ngrok http 8080
# smithery mcp publish "https://YOUR_SUBDOMAIN.ngrok-free.app" -n kioie/memex
```

## Deploy

Build from this directory or repo root:

```bash
go build -o server ./examples/http-deploy
MEMEX_DIR=/data ./server
```

### Render (native Go)

Use [`render.yaml`](./render.yaml) — root dir `examples/http-deploy`, health check `/health`.

Set `MEMEX_DIR` to a persistent disk path if your plan supports it.

### Fly.io

From repository root:

```bash
fly launch --config examples/http-deploy/fly.toml
fly volumes create memex_data --size 1 --region lhr
fly secrets set MEMEX_DIR=/data
fly deploy --config examples/http-deploy/fly.toml
```

Mount the volume at `/data` in `fly.toml` when ready for durable hosted memory.

## Environment

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | — | Set by PaaS (Render, Railway, Fly) |
| `MEMEX_HTTP_ADDR` | `:8080` | Listen address when `PORT` is unset |
| `MEMEX_DIR` | `~/.memex` | SQLite data directory — **set `/data` in containers** |
| `MEMEX_USER_ID` | `default` | Default memory scope |
| `MEMEX_HYBRID` | — | Set `1` for local vector fusion |
| `MEMEX_VERBOSE` | — | Log store path on startup |

## Publish on Smithery (not needed for default listing)

Smithery uses **MCPB stdio Docker** — see [docs/SMITHERY.md](../../docs/SMITHERY.md). Only publish an HTTP URL here if you self-host this example:

```bash
smithery mcp publish "https://YOUR_HOST" -n kioie/memex
```
