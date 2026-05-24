# Agent instructions (memex)

Guidance for AI coding agents working in this repository.

## Commands

- **Test all (CI / quick)**: `make test` or `go test -race -short ./...`
- **Test full scale**: `make test-full` or `go test -race ./memex ./cmd/memex`
- **Test integration**: `make test-integration` (MCP stdio subprocess; `-tags=integration`)
- **Benchmarks**: `go test -bench=. -benchmem ./memex`
- **Test package**: `go test -race -v ./memex`
- **Test single**: `go test -run TestName ./memex -v`
- **Coverage**: `make coverage` / `make coverage-check` (≥75% on `memex/`)
- **Lint**: `make lint` (requires [golangci-lint](https://golangci-lint.run/))
- **Vulnerabilities**: `make vulncheck` (requires [govulncheck](https://go.dev/doc/security/vuln/))
- **Build CLI**: `make build` → `bin/memex`
- **Install**: `make install` or `go install ./cmd/memex`
- **Run MCP server**: `go run ./cmd/memex serve` (stdio; for local MCP clients)

## Layout

- `memex/` — importable library (`github.com/kioie/memex/memex`): SQLite store + MCP tool registration
- `cmd/memex/` — CLI (`memex serve`)
- `integration/` — MCP stdio subprocess tests (`//go:build integration`)

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

## Test layout

| File | Covers |
|------|--------|
| `store_test.go` | Basic remember/recall/forget/list |
| `store_storage_test.go` | Field persistence, FTS search, limits, large payloads |
| `store_errors_test.go` | Validation, nil/closed store, query sanitization |
| `store_persistence_test.go` | Reopen durability, idempotent Open |
| `store_scale_test.go` | 100–5,000 inserts, search latency, concurrent writes, benchmarks |
| `store_security_test.go` | Special content, env paths, injection-style payloads |
| `mcp_roundtrip_test.go` | In-memory MCP client ↔ server tool roundtrip |
| `server_test.go` | MCP recall cap (50), error propagation, format helpers |

Integration (`integration/mcp_stdio_test.go`): real subprocess `memex serve` over stdio.

## CI workflows

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `unit.yml` | PR, push | Race + `-short`, release build, coverage gate |
| `integration.yml` | PR, push | MCP stdio roundtrip |
| `scale.yml` | Daily cron | Full scale suite (no `-short`) |
| `lint.yml` | PR, push | golangci-lint |
| `codeql.yml` | PR, push, weekly | Static security analysis |
| `security.yml` | PR, push, weekly | govulncheck |

**Known limits (current implementation):**

- **Recall default**: 10 rows at store level; MCP handler caps at **50**
- **Content size**: no app-level cap; SQLite TEXT supports ~1GB; tests verify 1KiB–1MiB
- **Scale**: 5,000 rows insert + FTS search passes; not a hard limit
- **Concurrent writes**: serialized with an in-process write mutex; SQLite `busy_timeout` (5s) covers multi-process access to the same DB file
- **FTS queries**: tokenized and quoted per word (`buildFTSQuery`); boolean operators in user input are not interpreted

## CI expectations

PRs should pass: unit tests, integration tests, lint, coverage check, release build. CodeQL and govulncheck run on PRs and weekly schedules.
