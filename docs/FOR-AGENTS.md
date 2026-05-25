# For AI agents

Instructions for LLM clients using memex tools. Fetch the MCP prompt **`memory_guide`** from the server for the same content at runtime.

## Your job

Help the user by **remembering durable facts** and **recalling them in later sessions** — without storing chat logs, secrets, or noise.

---

## When to save memory (`remember`)

Save **short, factual statements** the user would want repeated later:

- Preferences ("Prefers Go and table-driven tests")
- Decisions ("Use SQLite for local storage in this project")
- Project facts ("API base URL is https://api.example.com/v2")
- Your commitments ("Will add tests before merging")

**Do not save:**

- Passwords, API keys, tokens, or credentials
- Full conversation transcripts
- Temporary task state that expires when the chat ends
- Huge code blocks (summarize instead)

---

## When to read memory

| Situation | Tool |
|-----------|------|
| Starting work on a project | `retrieve_context` — load preferences within a token budget |
| You have keywords to search | `recall` — requires a query string |
| Browsing what's stored | `list_memories` — no query needed |
| Need one specific row by ID | `get_memory` |

**Prefer `retrieve_context`** when injecting memory into the model — it returns a bounded JSON payload (`max_tokens`, default 4096).

**Do not use `recall` without a query** — use `list_memories` instead.

---

## When to update or delete

| Situation | Tool |
|-----------|------|
| A fact changed | `update_memory` — supersedes the old row (new ID) |
| User asks to forget something | `forget` or `delete_memories` |
| User confirms wipe everything | `delete_all_memories` with `confirm=true` |

Do **not** call `remember` again for a revision — use `update_memory` on the existing ID.

---

## Good memory shape

One fact per call. Searchable wording. Optional tags and type.

```json
{
  "content": "User prefers PostgreSQL over MySQL for new services",
  "type": "preference",
  "tags": ["database", "architecture"],
  "source": "user"
}
```

**Types:** `note`, `preference`, `decision`, `fact`, `procedure`, `commitment`, `recommendation`, `action_taken`

**Source:** `user` (default), `agent` (your commitments/recommendations), `system`

---

## Session checklist

At the start of a coding session:

1. Fetch MCP prompt **`session_start`** (optional `run_id` argument)
2. Call **`retrieve_context`** with query like `"user preferences project conventions"`
3. Tag new memories with `run_id` when the user wants session grouping

---

## Scoping (multi-user / multi-agent)

- **`user_id`** — hard boundary between people or tenants (env: `MEMEX_USER_ID`)
- **`agent_id`** — separate agents sharing one user store (env: `MEMEX_AGENT_ID`)
- **`run_id`** — tag memories from one chat or CI run (env: `MEMEX_RUN_ID`)
- **`metadata`** — filter by project/repo keys on recall

Details: [examples/scoping](../examples/scoping/README.md)

---

## Built-in MCP prompts

| Prompt | Use |
|--------|-----|
| `memory_guide` | Full conventions — fetch when unsure |
| `session_start` | Beginning-of-session checklist |
| `remember_fact` | Turn draft text into a good `remember` call (arg: `draft`) |

---

## Outcome the user expects

After memex is wired correctly:

- They say something once → you remember it
- They return days later → you recall it without re-interviewing them
- They change their mind → you update the fact, history preserved
- Their data never leaves their machine unless they copy the database file
