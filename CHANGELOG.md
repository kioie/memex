# Changelog

All notable changes to memex are documented here. Version numbers follow [Semantic Versioning](https://semver.org/).

## [0.6.0] - 2026-05-24

### Added

- MCP prompts: `memory_guide`, `session_start`, `remember_fact` — agent conventions for when and how to use memory tools.
- `memex doctor` CLI — reports version, schema generation, database path, active memory counts, hybrid mode, and env defaults.
- `examples/` cookbook — Cursor MCP configs and multi-agent scoping guide.

## [0.5.1] - 2026-05-24

### Fixed

- CLI `memex version` now reports the same version as the MCP server (was stuck at `0.1.0`).

### Changed

- Single `memex.Version` constant shared by the library, MCP server, and CLI.
- Updated agent instructions (`AGENTS.md`) for v0.5.x tools and environment variables.

### Added

- Integration tests for `retrieve_context`, `MEMEX_HYBRID=1` recall, and supersession via `update_memory`.

## [0.5.0] - 2026-05-24

### Added

- Hybrid retrieval: FTS5 + entity boost + optional local vectors (`MEMEX_HYBRID=1`), fused with reciprocal rank fusion (RRF).
- Entity extraction and indexing on write; entity boost at query time.
- `retrieve_context` MCP tool (from v0.4.0) with greedy token-budget packing.

### Changed

- MCP server version `0.5.0`; 10 tools total.
- README updated for local-first positioning and v0.5.0 capabilities.

## [0.4.0] - 2026-05-24

### Added

- `retrieve_context` tool with configurable `max_tokens` (default 4096, max 32768).
- Greedy packing of ranked search hits within token budget.

### Changed

- `recall` requires a query; use `list_memories` to browse without keywords.

## [0.3.2] - 2026-05-24

### Added

- First-class `source` field: `user`, `agent`, or `system`.
- Agent-oriented memory types: `commitment`, `recommendation`, `action_taken`.
- Filter by `source` on `recall` and `list_memories`.

## [0.3.1] - 2026-05-24

### Added

- Append-only fact lifecycle: `update_memory` supersedes (new row), `forget` soft-deletes.
- `supersedes_id`, `valid_to` columns; default recall excludes inactive rows.
- `memory_events` append-only audit log.

## [0.3.0] - 2026-05-24

### Added

- Session scoping: `agent_id`, `run_id`, metadata equality filters.
- `MEMEX_AGENT_ID`, `MEMEX_RUN_ID` environment defaults.
- Scoped CRUD and history by `user_id`.

## [0.1.0] - 2026-05-24

### Added

- Initial release: local-first MCP memory server.
- Tools: `remember`, `recall`, `forget`, `get_memory`.
- SQLite + FTS5 storage under `~/.memex`.

[0.6.0]: https://github.com/kioie/memex/compare/v0.5.1...v0.6.0
[0.5.1]: https://github.com/kioie/memex/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/kioie/memex/releases/tag/v0.5.0
[0.4.0]: https://github.com/kioie/memex/compare/v0.3.2...v0.4.0
[0.3.2]: https://github.com/kioie/memex/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/kioie/memex/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/kioie/memex/compare/v0.1.0...v0.3.0
[0.1.0]: https://github.com/kioie/memex/releases/tag/v0.1.0
