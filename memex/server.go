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
const serverVersion = "0.1.0"

type rememberArgs struct {
	Content string   `json:"content" jsonschema:"Fact, preference, decision, or note to store (required)"`
	Tags    []string `json:"tags,omitempty" jsonschema:"Optional tags for filtering (e.g. user, project, preference)"`
	Type    string   `json:"type,omitempty" jsonschema:"Memory type: note, preference, decision, fact, or procedure"`
}

type recallArgs struct {
	Query string `json:"query,omitempty" jsonschema:"Search text; leave empty to list recent memories"`
	Limit int    `json:"limit,omitempty" jsonschema:"Maximum results (default 10, max 50)"`
}

type forgetArgs struct {
	ID string `json:"id" jsonschema:"Memory ID to delete (from remember or recall output)"`
}

type getArgs struct {
	ID string `json:"id" jsonschema:"Memory ID to fetch (required)"`
}

// NewMCPServer registers memex memory tools on a TinyServer backed by store.
func NewMCPServer(store *Store) (*tinymcp.TinyServer, error) {
	if store == nil {
		return nil, fmt.Errorf("store is required")
	}
	s := tinymcp.NewServer(serverName, serverVersion)
	h := &toolHandlers{store: store}

	if err := tinymcp.RegisterTool(s, "remember",
		"Store durable agent memory (preferences, decisions, facts). Use when the user or session reveals something worth recalling later. Do not use for transient chat filler — use recall (not remember) to read memories; use forget to remove outdated entries.",
		h.remember); err != nil {
		return nil, err
	}
	if err := tinymcp.RegisterTool(s, "recall",
		"Search or list stored memories by keyword. Use before answering questions about past preferences, decisions, or project context. Do not use for live tool/data lookups — use MCP tools for external systems; use remember (not recall) to save new facts.",
		h.recall); err != nil {
		return nil, err
	}
	if err := tinymcp.RegisterTool(s, "forget",
		"Delete a memory by ID when it is wrong or outdated. Use after recall surfaces the ID. Do not use to bulk-wipe memory — delete one ID at a time.",
		h.forget); err != nil {
		return nil, err
	}
	if err := tinymcp.RegisterTool(s, "get_memory",
		"Fetch one memory by exact ID. Use when recall returned an ID and you need the full content. Do not use for search — use recall instead.",
		h.getMemory); err != nil {
		return nil, err
	}
	return s, nil
}

type toolHandlers struct {
	store *Store
}

func (h *toolHandlers) remember(ctx context.Context, _ *mcp.CallToolRequest, args rememberArgs) (*mcp.CallToolResult, any, error) {
	mem, err := h.store.Remember(ctx, args.Content, args.Tags, args.Type)
	if err != nil {
		return nil, nil, err
	}
	return tinymcp.TextResult(formatRemembered(mem)), nil, nil
}

func (h *toolHandlers) recall(ctx context.Context, _ *mcp.CallToolRequest, args recallArgs) (*mcp.CallToolResult, any, error) {
	limit := args.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	memories, err := h.store.Recall(ctx, args.Query, limit)
	if err != nil {
		return nil, nil, err
	}
	return tinymcp.TextResult(formatRecall(memories, args.Query)), nil, nil
}

func (h *toolHandlers) forget(ctx context.Context, _ *mcp.CallToolRequest, args forgetArgs) (*mcp.CallToolResult, any, error) {
	if err := h.store.Forget(ctx, args.ID); err != nil {
		return nil, nil, err
	}
	return tinymcp.TextResult(fmt.Sprintf("Forgot memory %s.", args.ID)), nil, nil
}

func (h *toolHandlers) getMemory(ctx context.Context, _ *mcp.CallToolRequest, args getArgs) (*mcp.CallToolResult, any, error) {
	mem, err := h.store.Get(ctx, args.ID)
	if err != nil {
		return nil, nil, err
	}
	return tinymcp.TextResult(formatMemory(mem)), nil, nil
}

func formatRemembered(mem *Memory) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Remembered [%s] %s\n", mem.ID, mem.Content)
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
