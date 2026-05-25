package memex

import (
	"context"
	"fmt"
	"strings"

	"github.com/kioie/tiny-go-mcp-server/tinymcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const memoryGuideText = `# memex memory guide

Use memex for **durable facts** the user or agent should recall in future sessions — not for chat transcripts or secrets.

## When to remember
- User preferences, stack choices, naming conventions
- Decisions with rationale ("use SQLite, not Postgres, for this project")
- Agent commitments ("I will add tests before merging")
- Stable project facts (repo layout, deploy targets, API base URLs)

## When NOT to remember
- Ephemeral task chatter, partial code drafts, or full conversation logs
- Secrets (API keys, passwords, tokens) — never store credentials
- Duplicates: identical content for the same user_id returns the existing row

## Tool choice
| Goal | Tool |
|------|------|
| Inject bounded context into the model | retrieve_context (prefer max_tokens 2048–8192) |
| Inspect ranked search hits as text | recall (query required) |
| Browse recent rows without keywords | list_memories |
| Revise an existing fact | update_memory (supersedes; new ID) — not a second remember |
| Remove a fact | forget or delete_memories (soft-delete; audit kept) |

## Types and source
- Types: note, preference, decision, fact, procedure, commitment, recommendation, action_taken
- source=user for user-originated facts; source=agent for assistant commitments and recommendations
- commitment / recommendation / action_taken default source=agent when omitted

## Scoping
- user_id: tenant boundary (default from MEMEX_USER_ID)
- agent_id / run_id: isolate multi-agent or per-session memories (env: MEMEX_AGENT_ID, MEMEX_RUN_ID)
- metadata: exact key/value filters on recall and list_memories (e.g. project, repo)

## Retrieval tips
- Write atomic, searchable facts ("Prefers table-driven Go tests") not vague notes ("testing stuff")
- Set MEMEX_HYBRID=1 for local semantic + keyword fusion when paraphrases miss FTS
- At session start: retrieve_context for user preferences before planning work
`

const sessionStartTemplate = `# Session start checklist (memex)

Before coding, load durable context for this user/session.

1. Call retrieve_context with query "user preferences project conventions" and max_tokens around 4096.
2. Tag new remembers in this session with run_id=%q when provided.
3. Store agent commitments with type=commitment and source=agent.
4. Supersede outdated facts with update_memory instead of writing conflicting remembers.

If nothing relevant is stored yet, ask the user whether to persist preferences you infer during the session.
`

func registerMemoryPrompts(s *tinymcp.TinyServer) error {
	if err := tinymcp.RegisterPrompt(s,
		"memory_guide",
		"When to use memex remember/recall/retrieve_context, types, scoping, and anti-patterns. Fetch at session start or when unsure whether to write memory.",
		nil,
		handleMemoryGuidePrompt,
	); err != nil {
		return fmt.Errorf("register memory_guide prompt: %w", err)
	}
	if err := tinymcp.RegisterPrompt(s,
		"session_start",
		"Checklist for beginning a coding session: load preferences via retrieve_context, scope run_id, and write agent commitments correctly.",
		[]*mcp.PromptArgument{{
			Name:        "run_id",
			Description: "Optional run/session id to tag new memories (defaults to MEMEX_RUN_ID when unset)",
		}},
		handleSessionStartPrompt,
	); err != nil {
		return fmt.Errorf("register session_start prompt: %w", err)
	}
	if err := tinymcp.RegisterPrompt(s,
		"remember_fact",
		"Turn draft text into a well-formed memex fact before calling remember.",
		[]*mcp.PromptArgument{{
			Name:        "draft",
			Required:    true,
			Description: "Raw fact or note to distill into a durable memory",
		}},
		handleRememberFactPrompt,
	); err != nil {
		return fmt.Errorf("register remember_fact prompt: %w", err)
	}
	return nil
}

func handleMemoryGuidePrompt(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return tinymcp.PromptResult("memex memory conventions",
		tinymcp.UserPromptMessage(memoryGuideText),
	), nil
}

func handleSessionStartPrompt(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	runID := strings.TrimSpace(req.Params.Arguments["run_id"])
	if runID == "" {
		runID = ResolveRunID()
	}
	if runID == "" {
		runID = "(set run_id or MEMEX_RUN_ID)"
	}
	text := fmt.Sprintf(sessionStartTemplate, runID)
	return tinymcp.PromptResult("Session start with memex",
		tinymcp.UserPromptMessage(text),
	), nil
}

func handleRememberFactPrompt(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	draft := strings.TrimSpace(req.Params.Arguments["draft"])
	if draft == "" {
		return nil, fmt.Errorf("draft is required")
	}
	text := fmt.Sprintf(`Distill this draft into one atomic memex memory, then call remember:

Draft:
%s

Rules:
- One fact per remember call; keep under a few sentences
- Pick type (preference, decision, fact, commitment, etc.) and source (user vs agent)
- Add tags for project/domain filters
- Do not store secrets
- If revising an existing memory, use update_memory with the prior ID instead
`, draft)
	return tinymcp.PromptResult("Prepare a memex remember call",
		tinymcp.UserPromptMessage(text),
	), nil
}
