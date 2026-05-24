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
const serverVersion = "0.2.0"

type rememberArgs struct {
	Content  string         `json:"content" jsonschema:"Fact, preference, decision, or note to store (required)"`
	Tags     []string       `json:"tags,omitempty" jsonschema:"Optional tags for filtering (e.g. user, project, preference)"`
	Type     string         `json:"type,omitempty" jsonschema:"Memory type: note, preference, decision, fact, or procedure"`
	UserID   string         `json:"user_id,omitempty" jsonschema:"Scope memory to a user (defaults to MEMEX_USER_ID or default)"`
	Metadata map[string]any `json:"metadata,omitempty" jsonschema:"Optional JSON metadata bag (mem0-style)"`
}

type recallArgs struct {
	Query  string   `json:"query,omitempty" jsonschema:"Search text; leave empty to list recent memories"`
	Limit  int      `json:"limit,omitempty" jsonschema:"Maximum results (default 10, max 50)"`
	Offset int      `json:"offset,omitempty" jsonschema:"Skip first N results for pagination"`
	Tags   []string `json:"tags,omitempty" jsonschema:"Filter by tags (any match)"`
	Type   string   `json:"type,omitempty" jsonschema:"Filter by memory type"`
	UserID string   `json:"user_id,omitempty" jsonschema:"Scope search to user_id (defaults to MEMEX_USER_ID)"`
}

type listMemoriesArgs struct {
	Limit  int      `json:"limit,omitempty" jsonschema:"Maximum results (default 10, max 50)"`
	Offset int      `json:"offset,omitempty" jsonschema:"Skip first N results"`
	Tags   []string `json:"tags,omitempty" jsonschema:"Filter by tags (any match)"`
	Type   string   `json:"type,omitempty" jsonschema:"Filter by memory type"`
	UserID string   `json:"user_id,omitempty" jsonschema:"Scope list to user_id"`
}

type updateMemoryArgs struct {
	ID       string         `json:"id" jsonschema:"Memory ID to update (required)"`
	Content  string         `json:"content" jsonschema:"New content (required)"`
	Tags     []string       `json:"tags,omitempty" jsonschema:"Replace tags when provided"`
	Type     string         `json:"type,omitempty" jsonschema:"Replace memory type when provided"`
	Metadata map[string]any `json:"metadata,omitempty" jsonschema:"Replace metadata when provided"`
}

type forgetArgs struct {
	ID string `json:"id" jsonschema:"Memory ID to delete (from remember or recall output)"`
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
	ID string `json:"id" jsonschema:"Memory ID to fetch (required)"`
}

type memoryHistoryArgs struct {
	ID string `json:"id" jsonschema:"Memory ID to fetch history for (required)"`
}

// NewMCPServer registers memex memory tools on a tinymcp.TinyServer backed by store.
func NewMCPServer(store *Store) (*tinymcp.TinyServer, error) {
	if store == nil {
		return nil, fmt.Errorf("store is required")
	}
	s := tinymcp.NewServer(serverName, serverVersion)
	h := &toolHandlers{store: store}

	if err := tinymcp.RegisterTool(s, "remember",
		"Store durable agent memory (preferences, decisions, facts). Agent writes distilled facts directly (no server-side LLM infer). Duplicate content for the same user_id returns the existing memory. Use recall/search to read; use update_memory to revise; use forget/delete_memories to remove.",
		h.remember); err != nil {
		return nil, err
	}
	if err := tinymcp.RegisterTool(s, "recall",
		"Search or list stored memories (FTS5 keyword + BM25). Use before answering questions about past preferences or project context. Supports tags, type, user_id filters and pagination. Do not use for live external data — use MCP tools for that; use remember to save new facts.",
		h.recall); err != nil {
		return nil, err
	}
	if err := tinymcp.RegisterTool(s, "list_memories",
		"List memories with filters and pagination without a search query (mem0 get_memories). Use when browsing recent context; use recall when you have keywords.",
		h.listMemories); err != nil {
		return nil, err
	}
	if err := tinymcp.RegisterTool(s, "update_memory",
		"Overwrite a memory's content by ID. Use when a fact changed; do not add a duplicate via remember. Sibling: get_memory to read first.",
		h.updateMemory); err != nil {
		return nil, err
	}
	if err := tinymcp.RegisterTool(s, "get_memory",
		"Fetch one memory by exact ID. Use after recall/list returns an ID.",
		h.getMemory); err != nil {
		return nil, err
	}
	if err := tinymcp.RegisterTool(s, "forget",
		"Delete one memory by ID. Use delete_memories for batch; use delete_all_memories only with explicit user confirmation.",
		h.forget); err != nil {
		return nil, err
	}
	if err := tinymcp.RegisterTool(s, "delete_memories",
		"Delete multiple memories by ID list for a user scope (mem0 batch delete).",
		h.deleteMemories); err != nil {
		return nil, err
	}
	if err := tinymcp.RegisterTool(s, "delete_all_memories",
		"Delete all memories for a user_id scope. Requires confirm=true. Destructive — use rarely.",
		h.deleteAllMemories); err != nil {
		return nil, err
	}
	if err := tinymcp.RegisterTool(s, "memory_history",
		"Return ADD/UPDATE/DELETE audit trail for a memory ID (local mem0 history mirror).",
		h.memoryHistory); err != nil {
		return nil, err
	}
	return s, nil
}

type toolHandlers struct {
	store *Store
}

func (h *toolHandlers) remember(ctx context.Context, _ *mcp.CallToolRequest, args rememberArgs) (*mcp.CallToolResult, any, error) {
	opts := []RememberOption{WithUserID(args.UserID), WithMetadata(args.Metadata)}
	mem, err := h.store.Remember(ctx, args.Content, args.Tags, args.Type, opts...)
	if err != nil {
		return nil, nil, err
	}
	return tinymcp.TextResult(formatRemembered(mem)), nil, nil
}

func (h *toolHandlers) recall(ctx context.Context, _ *mcp.CallToolRequest, args recallArgs) (*mcp.CallToolResult, any, error) {
	limit, offset := clampLimitOffset(args.Limit, args.Offset)
	memories, err := h.store.Search(ctx, args.Query, MemoryFilter{
		UserID: args.UserID,
		Tags:   args.Tags,
		Type:   args.Type,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, nil, err
	}
	return tinymcp.TextResult(formatRecall(memories, args.Query)), nil, nil
}

func (h *toolHandlers) listMemories(ctx context.Context, _ *mcp.CallToolRequest, args listMemoriesArgs) (*mcp.CallToolResult, any, error) {
	limit, offset := clampLimitOffset(args.Limit, args.Offset)
	memories, err := h.store.List(ctx, MemoryFilter{
		UserID: args.UserID,
		Tags:   args.Tags,
		Type:   args.Type,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, nil, err
	}
	return tinymcp.TextResult(formatRecall(memories, "")), nil, nil
}

func (h *toolHandlers) updateMemory(ctx context.Context, _ *mcp.CallToolRequest, args updateMemoryArgs) (*mcp.CallToolResult, any, error) {
	mem, err := h.store.Update(ctx, args.ID, args.Content, args.Tags, args.Type, args.Metadata)
	if err != nil {
		return nil, nil, err
	}
	return tinymcp.TextResult(formatRemembered(mem)), nil, nil
}

func (h *toolHandlers) forget(ctx context.Context, _ *mcp.CallToolRequest, args forgetArgs) (*mcp.CallToolResult, any, error) {
	if err := h.store.Forget(ctx, args.ID); err != nil {
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
	mem, err := h.store.Get(ctx, args.ID)
	if err != nil {
		return nil, nil, err
	}
	return tinymcp.TextResult(formatMemory(mem)), nil, nil
}

func (h *toolHandlers) memoryHistory(ctx context.Context, _ *mcp.CallToolRequest, args memoryHistoryArgs) (*mcp.CallToolResult, any, error) {
	entries, err := h.store.History(ctx, args.ID)
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
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func formatRemembered(mem *Memory) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Remembered [%s] %s\n", mem.ID, mem.Content)
	if mem.UserID != "" && mem.UserID != defaultUserID {
		fmt.Fprintf(&b, "user_id: %s\n", mem.UserID)
	}
	if len(mem.Tags) > 0 {
		fmt.Fprintf(&b, "tags: %s\n", strings.Join(mem.Tags, ", "))
	}
	if mem.Type != "" && mem.Type != "note" {
		fmt.Fprintf(&b, "type: %s\n", mem.Type)
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
