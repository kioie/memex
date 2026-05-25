package memex

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kioie/tiny-go-mcp-server/tinymcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverName = "memex"
const serverVersion = "0.4.0"

type rememberArgs struct {
	Content  string         `json:"content" jsonschema:"Fact, preference, decision, or note to store (required)"`
	Tags     []string       `json:"tags,omitempty" jsonschema:"Optional tags for filtering (e.g. user, project, preference)"`
	Type     string         `json:"type,omitempty" jsonschema:"Memory type: note, preference, decision, fact, procedure, commitment, recommendation, or action_taken"`
	Source   string         `json:"source,omitempty" jsonschema:"Who originated the memory: user, agent, or system (defaults to user; agent types default to agent)"`
	UserID   string         `json:"user_id,omitempty" jsonschema:"Scope memory to a user (defaults to MEMEX_USER_ID or default)"`
	AgentID  string         `json:"agent_id,omitempty" jsonschema:"Scope memory to an agent (defaults to MEMEX_AGENT_ID when set)"`
	RunID    string         `json:"run_id,omitempty" jsonschema:"Tag memory with a run/session id (defaults to MEMEX_RUN_ID when set)"`
	Metadata map[string]any `json:"metadata,omitempty" jsonschema:"Optional JSON metadata bag"`
}

type recallArgs struct {
	Query    string            `json:"query" jsonschema:"Search text (required); use list_memories to browse without keywords"`
	Limit    int               `json:"limit,omitempty" jsonschema:"Maximum results (default 10, max 50)"`
	Offset   int               `json:"offset,omitempty" jsonschema:"Skip first N results for pagination"`
	Tags     []string          `json:"tags,omitempty" jsonschema:"Filter by tags (any match)"`
	Type     string            `json:"type,omitempty" jsonschema:"Filter by memory type"`
	Source   string            `json:"source,omitempty" jsonschema:"Filter by source: user, agent, or system"`
	UserID   string            `json:"user_id,omitempty" jsonschema:"Scope search to user_id (defaults to MEMEX_USER_ID)"`
	AgentID  string            `json:"agent_id,omitempty" jsonschema:"Filter by agent_id (defaults to MEMEX_AGENT_ID when set)"`
	RunID    string            `json:"run_id,omitempty" jsonschema:"Filter by run_id (defaults to MEMEX_RUN_ID when set)"`
	Metadata        map[string]string `json:"metadata,omitempty" jsonschema:"Exact metadata key/value filters (e.g. {\"source\":\"agent\"})"`
	IncludeInactive bool              `json:"include_inactive,omitempty" jsonschema:"Include superseded or deleted memories (default false)"`
}

type retrieveContextArgs struct {
	Query           string            `json:"query" jsonschema:"Search text (required)"`
	MaxTokens       int               `json:"max_tokens,omitempty" jsonschema:"Token budget for packed JSON output (default 4096, max 32768)"`
	Tags            []string          `json:"tags,omitempty" jsonschema:"Filter by tags (any match)"`
	Type            string            `json:"type,omitempty" jsonschema:"Filter by memory type"`
	Source          string            `json:"source,omitempty" jsonschema:"Filter by source: user, agent, or system"`
	UserID          string            `json:"user_id,omitempty" jsonschema:"Scope search to user_id (defaults to MEMEX_USER_ID)"`
	AgentID         string            `json:"agent_id,omitempty" jsonschema:"Filter by agent_id (defaults to MEMEX_AGENT_ID when set)"`
	RunID           string            `json:"run_id,omitempty" jsonschema:"Filter by run_id (defaults to MEMEX_RUN_ID when set)"`
	Metadata        map[string]string `json:"metadata,omitempty" jsonschema:"Exact metadata key/value filters"`
	IncludeInactive bool              `json:"include_inactive,omitempty" jsonschema:"Include superseded or deleted memories (default false)"`
}

type listMemoriesArgs struct {
	Limit    int               `json:"limit,omitempty" jsonschema:"Maximum results (default 10, max 50)"`
	Offset   int               `json:"offset,omitempty" jsonschema:"Skip first N results"`
	Tags     []string          `json:"tags,omitempty" jsonschema:"Filter by tags (any match)"`
	Type     string            `json:"type,omitempty" jsonschema:"Filter by memory type"`
	Source   string            `json:"source,omitempty" jsonschema:"Filter by source: user, agent, or system"`
	UserID   string            `json:"user_id,omitempty" jsonschema:"Scope list to user_id"`
	AgentID  string            `json:"agent_id,omitempty" jsonschema:"Filter by agent_id"`
	RunID    string            `json:"run_id,omitempty" jsonschema:"Filter by run_id"`
	Metadata        map[string]string `json:"metadata,omitempty" jsonschema:"Exact metadata key/value filters"`
	IncludeInactive bool              `json:"include_inactive,omitempty" jsonschema:"Include superseded or deleted memories (default false)"`
}

type updateMemoryArgs struct {
	ID       string         `json:"id" jsonschema:"Memory ID to update (required)"`
	Content  string         `json:"content" jsonschema:"New content (required)"`
	Tags     []string       `json:"tags,omitempty" jsonschema:"Replace tags when provided"`
	Type     string         `json:"type,omitempty" jsonschema:"Replace memory type when provided"`
	Source   string         `json:"source,omitempty" jsonschema:"Replace source when provided (user, agent, or system)"`
	Metadata map[string]any `json:"metadata,omitempty" jsonschema:"Replace metadata when provided"`
	UserID   string         `json:"user_id,omitempty" jsonschema:"Scope update to user_id (defaults to MEMEX_USER_ID)"`
	AgentID  string         `json:"agent_id,omitempty" jsonschema:"Scope update to agent_id when set"`
}

type forgetArgs struct {
	ID      string `json:"id" jsonschema:"Memory ID to delete (from remember or recall output)"`
	UserID  string `json:"user_id,omitempty" jsonschema:"Scope delete to user_id (defaults to MEMEX_USER_ID)"`
	AgentID string `json:"agent_id,omitempty" jsonschema:"Scope delete to agent_id when set"`
}

type deleteMemoriesArgs struct {
	IDs    []string `json:"ids" jsonschema:"Memory IDs to delete (required)"`
	UserID string   `json:"user_id,omitempty" jsonschema:"Scope deletes to user_id"`
}

type deleteAllMemoriesArgs struct {
	UserID  string `json:"user_id,omitempty" jsonschema:"User scope to wipe (defaults to MEMEX_USER_ID)"`
	Confirm bool   `json:"confirm" jsonschema:"Must be true to delete all memories in scope"`
}

type getArgs struct {
	ID      string `json:"id" jsonschema:"Memory ID to fetch (required)"`
	UserID  string `json:"user_id,omitempty" jsonschema:"Scope fetch to user_id (defaults to MEMEX_USER_ID)"`
	AgentID string `json:"agent_id,omitempty" jsonschema:"Scope fetch to agent_id when set"`
}

type memoryHistoryArgs struct {
	ID     string `json:"id" jsonschema:"Memory ID to fetch history for (required)"`
	UserID string `json:"user_id,omitempty" jsonschema:"Scope history to user_id (defaults to MEMEX_USER_ID)"`
}

// NewMCPServer registers memex memory tools on a tinymcp.TinyServer backed by store.
func NewMCPServer(store *Store) (*tinymcp.TinyServer, error) {
	if store == nil {
		return nil, fmt.Errorf("store is required")
	}
	s := tinymcp.NewServer(serverName, serverVersion)
	h := &toolHandlers{store: store}

	if err := tinymcp.RegisterTool(s, "remember",
		"Store durable agent memory (preferences, decisions, facts, commitments). Set source=agent for assistant-originated facts; commitment/recommendation/action_taken types default to agent. Duplicate content for the same user_id returns the existing active memory. Use recall or retrieve_context to read; use update_memory to supersede a fact; use forget/delete_memories to soft-delete.",
		h.remember); err != nil {
		return nil, err
	}
	if err := tinymcp.RegisterTool(s, "recall",
		"Search active stored memories by keyword (FTS5 + BM25). Query is required — use list_memories to browse recent rows without keywords. Superseded and deleted rows are excluded by default. Supports tags, type, source, user_id filters and pagination. For bounded context size use retrieve_context instead.",
		h.recall); err != nil {
		return nil, err
	}
	if err := tinymcp.RegisterTool(s, "retrieve_context",
		"Search ranked memories and pack JSON output within max_tokens (greedy, highest BM25 first). Requires query. Prefer over recall when the agent must stay within a token budget. Sibling: list_memories for browsing without a query.",
		h.retrieveContext); err != nil {
		return nil, err
	}
	if err := tinymcp.RegisterTool(s, "list_memories",
		"List memories with filters and pagination without a search query. Use when browsing recent context; use recall when you have keywords.",
		h.listMemories); err != nil {
		return nil, err
	}
	if err := tinymcp.RegisterTool(s, "update_memory",
		"Supersede an active memory by ID: closes the old row and appends a new one (returns a new ID). Prefer this over remember when revising an existing fact. Sibling: get_memory to read first.",
		h.updateMemory); err != nil {
		return nil, err
	}
	if err := tinymcp.RegisterTool(s, "get_memory",
		"Fetch one memory by exact ID. Use after recall/list returns an ID.",
		h.getMemory); err != nil {
		return nil, err
	}
	if err := tinymcp.RegisterTool(s, "forget",
		"Soft-delete one active memory by ID (sets valid_to; row kept for audit). Use delete_memories for batch; use delete_all_memories only with explicit user confirmation.",
		h.forget); err != nil {
		return nil, err
	}
	if err := tinymcp.RegisterTool(s, "delete_memories",
		"Delete multiple memories by ID list for a user scope.",
		h.deleteMemories); err != nil {
		return nil, err
	}
	if err := tinymcp.RegisterTool(s, "delete_all_memories",
		"Delete all memories for a user_id scope. Requires confirm=true. Destructive — use rarely.",
		h.deleteAllMemories); err != nil {
		return nil, err
	}
	if err := tinymcp.RegisterTool(s, "memory_history",
		"Return ADD/UPDATE/DELETE audit trail for a memory ID.",
		h.memoryHistory); err != nil {
		return nil, err
	}
	return s, nil
}

type toolHandlers struct {
	store *Store
}

func (h *toolHandlers) remember(ctx context.Context, _ *mcp.CallToolRequest, args rememberArgs) (*mcp.CallToolResult, any, error) {
	opts := []RememberOption{
		WithUserID(args.UserID),
		WithAgentID(args.AgentID),
		WithRunID(args.RunID),
		WithSource(args.Source),
		WithMetadata(args.Metadata),
	}
	mem, err := h.store.Remember(ctx, args.Content, args.Tags, args.Type, opts...)
	if err != nil {
		return nil, nil, err
	}
	return tinymcp.TextResult(formatRemembered(mem)), nil, nil
}

func (h *toolHandlers) recall(ctx context.Context, _ *mcp.CallToolRequest, args recallArgs) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(args.Query) == "" {
		return nil, nil, fmt.Errorf("recall requires a non-empty query; use list_memories to browse recent memories")
	}
	limit, offset := clampLimitOffset(args.Limit, args.Offset)
	memories, err := h.store.Search(ctx, args.Query, MemoryFilter{
		UserID:          args.UserID,
		AgentID:         args.AgentID,
		RunID:           args.RunID,
		Tags:            args.Tags,
		Type:            args.Type,
		Source:          args.Source,
		Metadata:        args.Metadata,
		IncludeInactive: args.IncludeInactive,
		Limit:           limit,
		Offset:          offset,
	})
	if err != nil {
		return nil, nil, err
	}
	return tinymcp.TextResult(formatRecall(memories, args.Query)), nil, nil
}

func (h *toolHandlers) retrieveContext(ctx context.Context, _ *mcp.CallToolRequest, args retrieveContextArgs) (*mcp.CallToolResult, any, error) {
	result, err := h.store.RetrieveContext(ctx, args.Query, MemoryFilter{
		UserID:          args.UserID,
		AgentID:         args.AgentID,
		RunID:           args.RunID,
		Tags:            args.Tags,
		Type:            args.Type,
		Source:          args.Source,
		Metadata:        args.Metadata,
		IncludeInactive: args.IncludeInactive,
	}, args.MaxTokens)
	if err != nil {
		return nil, nil, err
	}
	payload, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return tinymcp.TextResult(string(payload)), nil, nil
}

func (h *toolHandlers) listMemories(ctx context.Context, _ *mcp.CallToolRequest, args listMemoriesArgs) (*mcp.CallToolResult, any, error) {
	limit, offset := clampLimitOffset(args.Limit, args.Offset)
	memories, err := h.store.List(ctx, MemoryFilter{
		UserID:          args.UserID,
		AgentID:         args.AgentID,
		RunID:           args.RunID,
		Tags:            args.Tags,
		Type:            args.Type,
		Source:          args.Source,
		Metadata:        args.Metadata,
		IncludeInactive: args.IncludeInactive,
		Limit:           limit,
		Offset:          offset,
	})
	if err != nil {
		return nil, nil, err
	}
	return tinymcp.TextResult(formatRecall(memories, "")), nil, nil
}

func (h *toolHandlers) updateMemory(ctx context.Context, _ *mcp.CallToolRequest, args updateMemoryArgs) (*mcp.CallToolResult, any, error) {
	mem, err := h.store.Update(ctx, args.ID, UpdateInput{
		Content:  args.Content,
		Tags:     args.Tags,
		Type:     args.Type,
		Source:   args.Source,
		Metadata: args.Metadata,
		UserID:   args.UserID,
		AgentID:  args.AgentID,
	})
	if err != nil {
		return nil, nil, err
	}
	return tinymcp.TextResult(formatRemembered(mem)), nil, nil
}

func (h *toolHandlers) forget(ctx context.Context, _ *mcp.CallToolRequest, args forgetArgs) (*mcp.CallToolResult, any, error) {
	if err := h.store.Forget(ctx, args.ID, args.UserID, args.AgentID); err != nil {
		return nil, nil, err
	}
	return tinymcp.TextResult(fmt.Sprintf("Forgot memory %s.", args.ID)), nil, nil
}

func (h *toolHandlers) deleteMemories(ctx context.Context, _ *mcp.CallToolRequest, args deleteMemoriesArgs) (*mcp.CallToolResult, any, error) {
	n, err := h.store.ForgetBatch(ctx, args.IDs, args.UserID)
	if err != nil {
		return nil, nil, err
	}
	return tinymcp.TextResult(fmt.Sprintf("Deleted %d memories.", n)), nil, nil
}

func (h *toolHandlers) deleteAllMemories(ctx context.Context, _ *mcp.CallToolRequest, args deleteAllMemoriesArgs) (*mcp.CallToolResult, any, error) {
	if !args.Confirm {
		return nil, nil, fmt.Errorf("delete_all_memories requires confirm=true")
	}
	n, err := h.store.ForgetAll(ctx, args.UserID)
	if err != nil {
		return nil, nil, err
	}
	return tinymcp.TextResult(fmt.Sprintf("Deleted all %d memories in scope.", n)), nil, nil
}

func (h *toolHandlers) getMemory(ctx context.Context, _ *mcp.CallToolRequest, args getArgs) (*mcp.CallToolResult, any, error) {
	mem, err := h.store.Get(ctx, args.ID, args.UserID, args.AgentID)
	if err != nil {
		return nil, nil, err
	}
	return tinymcp.TextResult(formatMemory(mem)), nil, nil
}

func (h *toolHandlers) memoryHistory(ctx context.Context, _ *mcp.CallToolRequest, args memoryHistoryArgs) (*mcp.CallToolResult, any, error) {
	entries, err := h.store.History(ctx, args.ID, args.UserID)
	if err != nil {
		return nil, nil, err
	}
	payload, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return tinymcp.TextResult(string(payload)), nil, nil
}

func clampLimitOffset(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	return limit, max(0, offset)
}

func formatRemembered(mem *Memory) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Remembered [%s] %s\n", mem.ID, mem.Content)
	if mem.SupersedesID != "" {
		fmt.Fprintf(&b, "supersedes: %s\n", mem.SupersedesID)
	}
	if mem.UserID != "" && mem.UserID != defaultUserID {
		fmt.Fprintf(&b, "user_id: %s\n", mem.UserID)
	}
	if len(mem.Tags) > 0 {
		fmt.Fprintf(&b, "tags: %s\n", strings.Join(mem.Tags, ", "))
	}
	if mem.Type != "" && mem.Type != "note" {
		fmt.Fprintf(&b, "type: %s\n", mem.Type)
	}
	if mem.Source != "" && mem.Source != SourceUser {
		fmt.Fprintf(&b, "source: %s\n", mem.Source)
	}
	return strings.TrimSpace(b.String())
}

func formatRecall(memories []Memory, query string) string {
	if len(memories) == 0 {
		if strings.TrimSpace(query) == "" {
			return "No memories stored yet."
		}
		return fmt.Sprintf("No memories matched %q.", query)
	}
	payload, err := json.MarshalIndent(memories, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(payload)
}

func formatMemory(mem *Memory) string {
	payload, err := json.MarshalIndent(mem, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(payload)
}
