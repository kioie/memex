# Agent instructions (memex)

Guidance for AI coding agents working in this repository.

## Commands

- **Test all**: `make test` or `go test -race ./...`
- **Test package**: `go test -race -v ./memex`
- **Build CLI**: `make build` → `./memex`
- **Install**: `make install` or `go install ./cmd/memex`
- **Run MCP server**: `go run ./cmd/memex serve` (stdio; for local MCP clients)

## Layout

- `memex/` — importable library (`github.com/kioie/memex/memex`): SQLite store + MCP tool registration
- `cmd/memex/` — CLI (`memex serve`)

## MCP tools

| Tool | Purpose |
|------|---------|
| `remember` | Store a durable fact, preference, or decision |
| `recall` | Full-text search or list recent memories |
| `forget` | Delete one memory by ID |
| `get_memory` | Fetch one memory by exact ID |

Tool descriptions in `memex/server.go` follow MCP conventions: when to use, when not to, and sibling tools.

## Environment

- `MEMEX_DIR` — data directory (default: `~/.memex`, database at `memex.db`)
- `MEMEX_VERBOSE=1` — log store path to stderr (stdio is reserved for MCP)

## Code style

- Standard Go formatting (`gofmt` / `goimports`)
- Godoc on exported symbols
- Table-driven tests in `memex/*_test.go`

## CI expectations

PRs should pass `make test` and `make release`.
