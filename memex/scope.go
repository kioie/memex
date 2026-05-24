package memex

import (
	"os"
	"strings"
)

const defaultUserID = "default"

// ResolveUserID returns MEMEX_USER_ID or "default" when unset.
// Mirrors mem0's user_id scoping without requiring it on every tool call.
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
