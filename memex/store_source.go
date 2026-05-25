package memex

import (
	"fmt"
	"strings"
)

// Memory source values (who originated the fact).
const (
	SourceUser   = "user"
	SourceAgent  = "agent"
	SourceSystem = "system"
)

var allowedMemoryTypes = map[string]struct{}{
	"note":           {},
	"preference":     {},
	"decision":       {},
	"fact":           {},
	"procedure":      {},
	"commitment":     {},
	"recommendation": {},
	"action_taken":   {},
}

var agentFactTypes = map[string]struct{}{
	"commitment":     {},
	"recommendation": {},
	"action_taken":   {},
}

func validateMemoryType(memoryType string) error {
	if memoryType == "" {
		return nil
	}
	if _, ok := allowedMemoryTypes[memoryType]; !ok {
		return fmt.Errorf("invalid memory type %q", memoryType)
	}
	return nil
}

func validateSource(source string) error {
	switch source {
	case SourceUser, SourceAgent, SourceSystem:
		return nil
	default:
		return fmt.Errorf("invalid source %q: must be user, agent, or system", source)
	}
}

// resolveSource picks an explicit source or infers agent for commitment-style types.
func resolveSource(explicit, memoryType string) (string, error) {
	if s := strings.TrimSpace(explicit); s != "" {
		if err := validateSource(s); err != nil {
			return "", err
		}
		return s, nil
	}
	if _, ok := agentFactTypes[memoryType]; ok {
		return SourceAgent, nil
	}
	return SourceUser, nil
}

func normalizeStoredSource(source string) string {
	if strings.TrimSpace(source) == "" {
		return SourceUser
	}
	return source
}
