package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	handler := testHandler(t)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d", res.StatusCode)
	}
}

func TestServerCardEndpoint(t *testing.T) {
	handler := testHandler(t)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/.well-known/mcp/server-card.json")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("server-card status = %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("content-type = %q", ct)
	}
	if !strings.Contains(readBody(res), `"name": "memex"`) {
		t.Fatal("server-card missing memex name")
	}
}

func TestMCPInitialize(t *testing.T) {
	handler := testHandler(t)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}`
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("initialize status = %d body=%s", res.StatusCode, readBody(res))
	}
}

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	t.Setenv("MEMEX_DIR", filepath.Join(t.TempDir(), "memex"))
	t.Setenv("MEMEX_VERBOSE", "")
	handler, _, err := newHTTPMux()
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func readBody(res *http.Response) string {
	b, _ := io.ReadAll(res.Body)
	return string(b)
}
