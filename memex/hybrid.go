package memex

import (
	"os"
	"strings"
)

// HybridEnabled reports whether MEMEX_HYBRID=1 enables local vector retrieval.
func HybridEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MEMEX_HYBRID"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
