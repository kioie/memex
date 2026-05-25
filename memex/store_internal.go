package memex

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var errStoreClosed = errors.New("store is closed")

func trimRequired(v, field string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	return v, nil
}

func validateContent(content string) error {
	if content == "" {
		return errors.New("content is required")
	}
	if len(content) > maxMemoryContentLen {
		return fmt.Errorf("content exceeds maximum length of %d bytes", maxMemoryContentLen)
	}
	return nil
}

func contentHash(userID, content string) string {
	sum := sha256.Sum256([]byte(userID + "\x00" + strings.TrimSpace(content)))
	return hex.EncodeToString(sum[:])
}

func encodeMetadata(metadata map[string]any) (string, error) {
	if len(metadata) == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("encode metadata: %w", err)
	}
	return string(b), nil
}

func decodeMetadata(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return nil, nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("decode metadata: %w", err)
	}
	return out, nil
}
