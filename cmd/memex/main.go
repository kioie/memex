// Command memex is the memory extender CLI — local-first MCP server for AI agents.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/kioie/memex/memex"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "serve", "run":
			runServe()
			return
		case "version", "-v", "--version":
			fmt.Println(memex.Version)
			return
		case "help", "-h", "--help":
			printUsage()
			return
		}
	}
	runServe()
}

func runServe() {
	dir, err := memex.ResolveDir()
	if err != nil {
		log.Fatal(err)
	}
	store, err := memex.Open(dir)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	if os.Getenv("MEMEX_VERBOSE") != "" {
		log.Printf("memex store: %s", store.Path())
	}

	server, err := memex.NewMCPServer(store)
	if err != nil {
		log.Fatal(err)
	}
	if err := server.Start(); err != nil { // tinymcp stdio loop; stderr only for logs
		log.Fatal(err)
	}
}

func printUsage() {
	fmt.Println(`memex — memory extender for AI agents

Usage:
  memex [command]

Commands:
  serve     Start the MCP server over stdio (default)
  version   Print version
  help      Show this help

Environment:
  MEMEX_DIR       Data directory (default: ~/.memex)
  MEMEX_HYBRID=1  Enable local vector retrieval (fused with keyword + entity signals)
  MEMEX_VERBOSE   Log store path to stderr`)
}
