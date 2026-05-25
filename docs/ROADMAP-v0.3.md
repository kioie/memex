# memex v0.3 roadmap

Local-first memory with **accurate recall under a token budget**, **append-only facts**, **hybrid retrieval**, and **strong scoping** — without cloud LLMs or mandatory vector DBs.

## Current baseline

| Area | Status (v0.3.1 / Phase 2) |
|------|---------------------------|
| Scoping | `user_id`, `agent_id`, `run_id`; scoped get/forget/update/history; 256 KiB cap |
| Retrieval | FTS5 + BM25; active-only by default; optional `include_inactive` |
| Writes | Hash dedup; `update_memory` supersedes (new row); soft-delete via `valid_to` |
| History | `memory_history` audit + `memory_events` append-only log |
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

**Goal:** Close cross-tenant leaks with `user_id` / `agent_id` / `run_id` + metadata partitions.

- [x] `agent_id`, `run_id` columns + migration
- [x] `MEMEX_AGENT_ID`, `MEMEX_RUN_ID` env defaults
- [x] Metadata equality filters on `recall` / `list_memories`
- [x] Scope `update_memory` and `memory_history` by `user_id`
- [x] Optional `agent_id` on ID-based tools when strict match required

**Branch:** `kioie/v03-phase1-scoping`

## Phase 2 — ADD-only facts + history ✅ (v0.3.1)

**Goal:** Append facts instead of reconciling with UPDATE/DELETE at write time.

- [x] `supersedes_id`, `valid_to` columns
- [x] Default recall excludes superseded rows
- [x] `memory_events` append-only log
- [x] Deprecate destructive updates as primary path (`update_memory` supersedes; `forget` soft-deletes)

**Branch:** `kioie/v03-phase2-add-only`

## Phase 3 — Agent facts first-class

**Goal:** Reliable recall of agent commitments and assistant-originated facts.

- `source`: `user` | `agent` | `system`
- Types: `commitment`, `recommendation`, `action_taken`
- Filter by `source` on recall

## Phase 4 — Token-efficient retrieval

**Goal:** Bounded token output per retrieval call (configurable budget).

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
- Large-scale external evaluation harnesses
