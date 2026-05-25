// Deployable streamable HTTP MCP for Smithery URL listing and remote try-it flows.
//
// End users connect via Smithery or any streamable HTTP MCP client — no local
// binary required. You host this service on HTTPS (Render, Railway, Fly, etc.).
//
// Run locally:
//
//	MEMEX_DIR=./data go run ./examples/http-deploy
//
// Publish to Smithery (after HTTPS is live):
//
//	smithery auth login
//	smithery mcp publish "https://YOUR_HOST" -n kioie/memex
//
// See examples/http-deploy/README.md and docs/SMITHERY.md.
package main

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/kioie/memex/memex"
	"github.com/kioie/tiny-go-mcp-server/tinymcp"
)

//go:embed server-card.json
var serverCardJSON []byte

func main() {
	addr := listenAddr()
	handler, storePath, err := newHTTPMux()
	if err != nil {
		log.Fatal(err)
	}
	if os.Getenv("MEMEX_VERBOSE") != "" {
		log.Printf("memex HTTP listening on %s (MCP at /, store %s)", addr, storePath)
	}
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}

func newHTTPMux() (http.Handler, string, error) {
	dir, err := memex.ResolveDir()
	if err != nil {
		return nil, "", err
	}
	store, err := memex.Open(dir)
	if err != nil {
		return nil, "", err
	}

	server, err := memex.NewMCPServer(store)
	if err != nil {
		store.Close()
		return nil, "", err
	}

	mcpHandler, err := tinymcp.StreamableHTTPHandler(server, &tinymcp.HTTPOptions{
		Stateless:                  true,
		DisableLocalhostProtection: true,
	})
	if err != nil {
		store.Close()
		return nil, "", fmt.Errorf("streamable HTTP handler: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	mux.Handle("/.well-known/mcp/server-card.json", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(serverCardJSON)
	}))
	mux.Handle("/", mcpHandler)

	return mux, store.Path(), nil
}

func listenAddr() string {
	if port := os.Getenv("PORT"); port != "" {
		return ":" + port
	}
	if v := os.Getenv("MEMEX_HTTP_ADDR"); v != "" {
		return v
	}
	return ":8080"
}
