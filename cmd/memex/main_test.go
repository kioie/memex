package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kioie/memex/memex"
)

func TestCLIVersion(t *testing.T) {
	out := runCLI(t, "version")
	if strings.TrimSpace(out) != memex.Version {
		t.Fatalf("version output = %q, want %q", out, memex.Version)
	}
}

func TestCLIHelp(t *testing.T) {
	out := runCLI(t, "help")
	for _, want := range []string{"memex", "serve", "doctor", "MEMEX_DIR"} {
		if !strings.Contains(out, want) {
			t.Fatalf("help missing %q:\n%s", want, out)
		}
	}
}

func TestCLIDoctor(t *testing.T) {
	dir := t.TempDir()
	bin := buildCLIBinary(t)
	cmd := exec.Command(bin, "doctor")
	cmd.Env = append(os.Environ(), "MEMEX_DIR="+dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("memex doctor: %v\n%s", err, out)
	}
	text := string(out)
	for _, want := range []string{"memex doctor", "schema:", "database:", "status:      ok"} {
		if !strings.Contains(text, want) {
			t.Fatalf("doctor output missing %q:\n%s", want, text)
		}
	}
}

func runCLI(t *testing.T, args ...string) string {
	t.Helper()
	bin := buildCLIBinary(t)
	cmd := exec.Command(bin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("memex %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func buildCLIBinary(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	dir := t.TempDir()
	bin := filepath.Join(dir, "memex")
	if os.Getenv("GOOS") == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/memex")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build memex cli: %v\n%s", err, out)
	}
	return bin
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
