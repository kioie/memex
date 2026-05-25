package memex

import (
	"strings"
	"testing"
)

func TestFilterClausesSQLUsesBoundParameters(t *testing.T) {
	args := []any{"user-1"}
	sql, err := filterClausesSQL(MemoryFilter{Type: "note", Tags: []string{"go", `%evil%`}}, &args, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sql, clauseFilterMemoryType) || !strings.Contains(sql, clauseFilterTag) {
		t.Fatalf("filter SQL = %q", sql)
	}
	if len(args) != 4 {
		t.Fatalf("args = %v, want 4 bound values", args)
	}
	if args[1] != "note" {
		t.Fatalf("type arg = %v", args[1])
	}
	for _, arg := range args[2:] {
		pattern, ok := arg.(string)
		if !ok || strings.Contains(pattern, "%evil%") {
			t.Fatalf("tag pattern not sanitized: %v", arg)
		}
	}
}

func TestResolveDataDirRejectsEmpty(t *testing.T) {
	_, err := resolveDataDir("   ")
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("resolveDataDir() error = %v", err)
	}
}

func TestSanitizeFTSTokenStripsOperators(t *testing.T) {
	if got := sanitizeFTSToken(`OR*injection"`); got != "ORinjection" {
		t.Fatalf("sanitizeFTSToken = %q", got)
	}
}
