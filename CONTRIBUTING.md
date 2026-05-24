# Contributing

Thank you for contributing to **memex**.

## Prerequisites

- Go 1.26+ ([download](https://go.dev/dl/))
- Optional: [golangci-lint](https://golangci-lint.run/) for local linting

## Setup

```bash
git clone https://github.com/kioie/memex.git
cd memex
go mod download
```

## Test tiers

memex uses a tiered test strategy (similar to large OSS projects): fast checks on every PR, deeper suites on a schedule.

| Tier | Command | When | What it covers |
|------|---------|------|----------------|
| **Unit** | `make test` | Every PR | Race detector + `-short` (skips 1k/5k scale, 1MiB payloads) |
| **Full scale** | `make test-full` | Nightly + before releases | 1,000–5,000 row inserts, large payloads, concurrency |
| **Integration** | `make test-integration` | PR + nightly | Real MCP stdio subprocess roundtrip |
| **Coverage** | `make coverage` | CI | Package coverage report |
| **Security** | `make vulncheck` | Weekly CI | Go vulnerability scan |
| **SonarQube** | CI workflow `sonar.yml` | PR + push to `main` | Static analysis + 80% coverage floor + strict quality gate |

## SonarQube setup

1. Import the repo at [SonarCloud](https://sonarcloud.io) (organization `kioie`, project key `kioie_memex`).
2. GitHub **Settings → Secrets and variables → Actions**:
   - Secret: `SONAR_TOKEN` (from SonarCloud)
   - Variable: `SONAR_HOST_URL` = `https://sonarcloud.io`
3. Create or clone a **strict quality gate** in SonarCloud and assign it to this project:

   | Condition | Scope | Threshold |
   |-----------|-------|-----------|
   | No new bugs | New code | 0 |
   | No new vulnerabilities | New code | 0 |
   | No new code smells / maintainability rating | New code | A (or 0 smells) |
   | Coverage | New code | ≥ 80% |
   | Duplicated lines | New code | ≤ 3% |
   | Security hotspots reviewed | New code | 100% |
   | Coverage | Overall | ≥ 75% |

4. **Disable the quality gate fudge factor** for this project (Project Settings → Quality Gate) so small PRs still enforce coverage and duplication rules.
5. Use the **Sonar way** quality profile (default) or stricter; do not relax issue severities for Go.

Local check before pushing: `make coverage-sonar-check` (80% floor on `memex/`).

## Workflow

1. Create a branch from `main` (use prefix `kioie/` if you maintain this repo).
2. Make changes in `memex/` (library) or `cmd/memex/` (CLI).
3. Run checks locally:

   ```bash
   make test
   make test-integration   # if MCP or CLI changed
   make lint               # if golangci-lint is installed
   ```

4. Open a pull request against `main`.

## Coverage expectation

New code in `memex/` should maintain **≥75%** package coverage (`make coverage-check` enforces this in CI). Prefer meaningful tests over trivial assertions.

## Adding MCP tools

1. Add store logic in `memex/store.go` if needed.
2. Register the tool in `memex/server.go` with a description that states when to use, when not to, and sibling tools.
3. Add unit tests in `memex/*_test.go` and MCP roundtrip coverage in `memex/mcp_roundtrip_test.go`.
4. Update `AGENTS.md` tool table.

## Questions

Open a [GitHub issue](https://github.com/kioie/memex/issues) for bugs or feature requests.
