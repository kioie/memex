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
- **Sonar coverage**: `make coverage-sonar` / `make coverage-sonar-check` (≥80% on `memex/`, used by SonarQube CI)
- **Lint**: `make lint` (requires [golangci-lint](https://golangci-lint.run/))
- **Vulnerabilities**: `make vulncheck` (requires [govulncheck](https://go.dev/doc/security/vuln/))
- **Build CLI**: `make build` → `bin/memex`
- **Install**: `make install` or `go install ./cmd/memex`
- **Run MCP server**: `go run ./cmd/memex serve` (stdio; for local MCP clients)
- **Doctor**: `memex doctor` — store path, schema version, memory counts, hybrid flag

## Layout

- `memex/` — importable library (`github.com/kioie/memex/memex`): SQLite store + MCP tool registration
- `docs/` — user-facing guides ([Getting started](docs/GETTING-STARTED.md), [For AI agents](docs/FOR-AGENTS.md))
- `examples/` — Cursor MCP configs and scoping cookbook

## MCP layer

| Concern | Where | Notes |
|---------|-------|-------|
| Store / FTS | `memex/store.go` | No MCP imports |
| Hybrid / entities | `memex/store_hybrid.go`, `store_entities.go`, `store_embed.go` | RRF fusion; `MEMEX_HYBRID=1` |
| Tool handlers | `memex/server.go` | `tinymcp.RegisterTool`, `tinymcp.TextResult` |
| MCP prompts | `memex/prompts.go` | `memory_guide`, `session_start`, `remember_fact` |
| Stdio serve | `cmd/memex/main.go` | `server.Start()` from tinymcp |
| In-memory tests | `mcp_roundtrip_test.go` | `server.RawServer().Connect(...)` |
| Subprocess tests | `integration/mcp_stdio_test.go` | Full CLI + stdio path |

Handler signature: `func(ctx, *mcp.CallToolRequest, args In) (*mcp.CallToolResult, any, error)` with struct tags `json` + `jsonschema`. Tool descriptions: when to use, when not to, sibling tools — same pattern as other tinymcp servers. Reference: [tinymcp package docs](https://pkg.go.dev/github.com/kioie/tiny-go-mcp-server/tinymcp).

## MCP tools

| Tool | Purpose |
|------|---------|
| `remember` | Store a durable fact (hash dedup per user_id; optional agent/run scope, source, metadata) |
| `recall` | FTS + hybrid search; **query required** — use `list_memories` to browse |
| `retrieve_context` | Ranked search packed within `max_tokens` (greedy; hybrid fusion) |
| `list_memories` | Paginated list with filters (no query) |
| `update_memory` | Supersede an active memory by ID (append-only; returns new ID) |
| `get_memory` | Fetch one memory by exact ID (scoped; optional agent_id) |
| `forget` | Soft-delete one memory (scoped; optional agent_id) |
| `delete_memories` | Batch soft-delete |
| `delete_all_memories` | Scoped wipe (`confirm=true`) |
| `memory_history` | ADD / supersede / delete audit trail (scoped by user_id) |

Memory types: `note`, `preference`, `decision`, `fact`, `procedure`, `commitment`, `recommendation`, `action_taken`.

Tool descriptions in `memex/server.go` follow MCP conventions: when to use, when not to, and sibling tools.

## Environment

- `MEMEX_DIR` — data directory (default: `~/.memex`, database at `memex.db`)
- `MEMEX_USER_ID` — default user scope for memories (default `default`)
- `MEMEX_AGENT_ID` — default agent scope when tool args omit `agent_id`
- `MEMEX_RUN_ID` — default run/session tag when tool args omit `run_id`
- `MEMEX_HYBRID=1` — enable local vector retrieval (deterministic embeddings, fused with FTS + entities via RRF)
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
| `store_phase*.go` / `*_test.go` | Phase 3–5: source, retrieve_context, hybrid RRF |
| `mcp_roundtrip_test.go` | In-memory MCP client ↔ server tool roundtrip |
| `server_test.go` | MCP recall cap (50), error propagation, format helpers |

Integration (`integration/mcp_stdio_test.go`): real subprocess `memex serve` over stdio — remember/recall, `retrieve_context`, hybrid mode, supersession, MCP prompts.

## CI workflows

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `unit.yml` | PR, push | Race + `-short`, release build, coverage gate |
| `integration.yml` | PR, push | MCP stdio roundtrip |
| `scale.yml` | Daily cron | Full scale suite (no `-short`) |
| `lint.yml` | PR, push | golangci-lint |
| `codeql.yml` | PR, push, weekly | Static security analysis |
| `security.yml` | PR, push, weekly | govulncheck |
| `sonar.yml` | PR, push | SonarQube analysis, 80% coverage floor, strict quality gate |

## SonarQube setup

1. Import the repo at [SonarCloud](https://sonarcloud.io) (org `kioie`, project key `kioie_memex`).
2. GitHub **Settings → Secrets and variables → Actions**:
   - Secret: **`SONAR_TOKEN`** (required — from SonarCloud → My Account → Security)
   - Variable: `SONAR_HOST_URL` (optional — only for self-hosted; defaults to `https://sonarcloud.io`)
3. Assign a **strict quality gate** to the project (see CONTRIBUTING.md). CI fails on `FAILED` or `WARN`.

**Stringent CI behavior (`sonar.yml`):**

- Pre-scan coverage floor: **80%** on `memex/` (`make coverage-sonar-check`)
- Quality gate poll timeout: **600s**
- **WARN and FAILED** both fail the workflow (only `PASSED` accepted)
- New code measured against `main` (`sonar.newCode.referenceBranch`)

**Known limits (current implementation):**

- **Recall default**: 10 rows at store level; MCP handler caps at **50**
- **Content size**: max **256 KiB** per memory (`remember` / `update_memory`); SQLite TEXT supports larger values but memex rejects oversize writes
- **Scale**: 5,000 rows insert + FTS search passes; not a hard limit
- **Hybrid vectors**: full-user embedding scan in Go (no sqlite-vec ANN yet)
- **Concurrent writes**: serialized with an in-process write mutex; SQLite `busy_timeout` (5s) covers multi-process access to the same DB file
- **FTS queries**: tokenized and quoted per word (`buildFTSQuery`); boolean operators in user input are not interpreted

## CI expectations

PRs should pass: unit tests, integration tests, lint, coverage check, release build, SonarQube quality gate. CodeQL and govulncheck run on PRs and weekly schedules.
