// Command confluence-cli is a small, dependency-free CLI for reading and
// writing pages on a self-hosted Confluence Server/Data Center instance.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

// version is overridable at build time:
//
//	go build -ldflags "-X main.version=1.2.3"
var version = "0.1.0-dev"

func main() {
	os.Exit(run(os.Args[1:]))
}

// run dispatches a command and returns the process exit code. It is split out
// from main so it can be exercised by tests.
func run(argv []string) int {
	if len(argv) < 1 {
		usage(os.Stderr)
		return 2
	}

	cmd, args := argv[0], argv[1:]
	var err error
	switch cmd {
	case "search":
		err = runSearch(args)
	case "get":
		err = runGet(args)
	case "create":
		err = runCreate(args)
	case "update":
		err = runUpdate(args)
	case "comment":
		err = runComment(args)
	case "spaces":
		err = runSpaces(args)
	case "tree":
		err = runTree(args)
	case "children":
		err = runChildren(args)
	case "move":
		err = runMove(args)
	case "labels":
		err = runLabels(args)
	case "delete":
		err = runDelete(args)
	case "generate-skill":
		err = runGenerateSkill(args)
	case "version", "-v", "--version":
		fmt.Printf("confluence-cli %s\n", version)
	case "help", "-h", "--help":
		usage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		usage(os.Stderr)
		return 2
	}

	if err != nil {
		// `-h`/`--help` on a subcommand surfaces flag.ErrHelp after the
		// command already printed its usage; treat it as a clean exit.
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

func usage(w *os.File) {
	fmt.Fprintf(w, `confluence-cli %s — CLI for self-hosted Confluence (Server/Data Center)

Usage:
  confluence-cli <command> [flags]

Commands:
  search    Search pages with CQL or a text query
  get       Fetch a page by ID
  create    Create a new page (use --parent to create a child)
  update    Update an existing page (auto-increments version)
  comment   Add a comment to a page
  delete    Delete a page (asks for confirmation)
  spaces    List spaces
  tree      Print the page tree of a space or subtree
  children  List the direct child pages of a page
  move      Move / re-parent / promote / demote a page
  labels    List, add, or remove labels on a page
  generate-skill  Write a confluence-skill.md for an AI agent
                  (flavors: claude, codex, gemini, opencode; none = generic)
  version   Print version
  help      Show this help

Run "confluence-cli <command> -h" for command-specific flags.

Authentication (Server/Data Center):
  Personal Access Token (preferred):  --token / CONFLUENCE_TOKEN
  Basic auth:                          --user + --password / CONFLUENCE_USER + CONFLUENCE_PASSWORD
  Site base URL:                       --base-url / CONFLUENCE_BASE_URL
  Default space (search/create):       --space / CONFLUENCE_SPACE
`, version)
}
