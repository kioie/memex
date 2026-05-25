package memex

import (
	"os"
	"strings"
)

const defaultUserID = "default"

// ResolveUserID returns MEMEX_USER_ID or "default" when unset.
func ResolveUserID() string {
	if id := strings.TrimSpace(os.Getenv("MEMEX_USER_ID")); id != "" {
		return id
	}
	return defaultUserID
}

// ResolveUserIDArg picks an explicit user_id from a tool argument, else ResolveUserID.
func ResolveUserIDArg(explicit string) string {
	if id := strings.TrimSpace(explicit); id != "" {
		return id
	}
	return ResolveUserID()
}

// ResolveAgentID returns MEMEX_AGENT_ID or empty when unset.
func ResolveAgentID() string {
	return strings.TrimSpace(os.Getenv("MEMEX_AGENT_ID"))
}

// ResolveAgentIDArg picks an explicit agent_id from a tool argument, else ResolveAgentID.
func ResolveAgentIDArg(explicit string) string {
	if id := strings.TrimSpace(explicit); id != "" {
		return id
	}
	return ResolveAgentID()
}

// ResolveRunID returns MEMEX_RUN_ID or empty when unset.
func ResolveRunID() string {
	return strings.TrimSpace(os.Getenv("MEMEX_RUN_ID"))
}

// ResolveRunIDArg picks an explicit run_id from a tool argument, else ResolveRunID.
func ResolveRunIDArg(explicit string) string {
	if id := strings.TrimSpace(explicit); id != "" {
		return id
	}
	return ResolveRunID()
}
