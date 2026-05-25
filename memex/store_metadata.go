package memex

import (
	"fmt"
	"regexp"
	"strings"
)

var metadataKeyPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func sanitizeMetadataKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" || !metadataKeyPattern.MatchString(key) {
		return "", fmt.Errorf("invalid metadata filter key: %q", key)
	}
	return key, nil
}

func metadataFilterClause(colPrefix, key string) (string, error) {
	key, err := sanitizeMetadataKey(key)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(` AND json_extract(%smetadata, '$.%s') = ?`, colPrefix, key), nil
}
