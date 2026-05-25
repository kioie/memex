# memex v0.3 roadmap

Local-first memory inspired by [mem0 research](https://mem0.ai/research): **accurate recall under a token budget**, **append-only facts**, **hybrid retrieval**, and **strong scoping** — without cloud LLMs or mandatory vector DBs.

## Current baseline

| Area | Status (v0.3.0 / Phase 1) |
|------|---------------------------|
| Scoping | `user_id`, `agent_id`, `run_id`; scoped get/forget/update/history; 256 KiB cap |
| Retrieval | FTS5 + BM25; filters: tags, type, agent/run, metadata equality |
| Writes | Hash dedup; `update_memory` scoped by caller `user_id` |
| History | Per-`memory_id` audit trail; scoped by `user_id` |
| Agent facts | `agent_id` / `run_id` stored; `MEMEX_AGENT_ID` / `MEMEX_RUN_ID` env defaults |
| Metadata | Stored and filterable on `recall` / `list_memories` |

## Target architecture

```
Write:  MCP remember → validate → append row → FTS + (future: entities/vectors)
Read:   recall / retrieve_context → token budget → hybrid rank → JSON output
```

## Release phases

| Version | Phase | Theme |
|---------|-------|-------|
| **v0.3.0** | 1 | Session scoping (`agent_id`, `run_id`, metadata filters) |
| **v0.3.1** | 2 | ADD-only facts + supersession |
| **v0.3.2** | 3 | Agent facts first-class (`source`, commitment types) |
| **v0.4.0** | 4 | Token-efficient `retrieve_context` |
| **v0.5.0** | 5 | Multi-signal retrieval (entities + optional sqlite-vec) |

---

## Phase 1 — User / agent scoping ✅ (v0.3.0)

**Goal:** Close cross-tenant leaks; mem0-style `user_id` / `agent_id` / `run_id` + metadata partitions.

- [x] `agent_id`, `run_id` columns + migration
- [x] `MEMEX_AGENT_ID`, `MEMEX_RUN_ID` env defaults
- [x] Metadata equality filters on `recall` / `list_memories`
- [x] Scope `update_memory` and `memory_history` by `user_id`
- [x] Optional `agent_id` on ID-based tools when strict match required

**Branch:** `kioie/v03-phase1-scoping`

## Phase 2 — ADD-only facts + history

**Goal:** Append facts instead of reconciling with UPDATE/DELETE at write time.

- `supersedes_id`, `valid_to` columns
- Default recall excludes superseded rows
- `memory_events` append-only log
- Deprecate destructive updates as primary path

## Phase 3 — Agent facts first-class

**Goal:** Reliable recall of agent commitments (LongMemEval assistant category).

- `source`: `user` | `agent` | `system`
- Types: `commitment`, `recommendation`, `action_taken`
- Filter by `source` on recall

## Phase 4 — Token-efficient retrieval

**Goal:** Mem0-style ~7k token target per retrieval call.

- New MCP tool: `retrieve_context` with `max_tokens`
- Greedy pack ranked results
- Strict split: `recall` requires query; browse via `list_memories`

## Phase 5 — Multi-signal retrieval

**Goal:** Fuse keyword + entity + (optional) semantic signals locally.

- Entity extraction table + query boost
- Optional `sqlite-vec` behind `MEMEX_HYBRID=1`
- Reciprocal rank fusion before token packing

---

## Success metrics

| Metric | Target |
|--------|--------|
| Cross-scope leaks | 0 in isolation test suite |
| Tokens per recall (Phase 4) | p95 < 8k via `retrieve_context` |
| Recall latency at 5k rows | p95 < 50ms (existing scale tests) |

## Out of scope

- LLM extraction pipeline
- Cloud embeddings API
- BEAM-scale 10M evaluation harness
