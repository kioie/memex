package memex

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// resolveDataDir validates and returns an absolute path for memex storage.
func resolveDataDir(dir string) (string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "", errors.New("memex dir is required")
	}
	dir = filepath.Clean(dir)
	if dir == "." {
		return "", errors.New("memex dir is required")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve memex dir: %w", err)
	}
	return abs, nil
}

// ResolveDir picks MEMEX_DIR (validated absolute path), or falls back to DefaultDir.
func ResolveDir() (string, error) {
	if dir := strings.TrimSpace(os.Getenv("MEMEX_DIR")); dir != "" {
		return resolveDataDir(dir)
	}
	return DefaultDir()
}
