package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

// skillFlavors are the supported agent platforms. The empty string and
// "generic" both map to the universal, frontmatter-less skill.
var skillFlavors = map[string]bool{
	"":         true, // generic
	"generic":  true,
	"claude":   true,
	"codex":    true,
	"gemini":   true,
	"opencode": true,
}

func runGenerateSkill(args []string) error {
	// The flavor is an optional leading positional, e.g.
	//   generate-skill            -> generic
	//   generate-skill claude     -> claude
	//   generate-skill claude --stdout
	var flavor string
	rest := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		flavor, rest = args[0], args[1:]
	}

	fs := flag.NewFlagSet("generate-skill", flag.ExitOnError)
	var (
		out   = fs.String("out", "confluence-skill.md", "output file path")
		toOut = fs.Bool("stdout", false, "write to stdout instead of a file")
		force = fs.Bool("force", false, "overwrite the output file if it exists")
	)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: confluence-cli generate-skill [flavor] [flags]")
		fmt.Fprintf(fs.Output(), "\nFlavors: %s\n", strings.Join(flavorNames(), ", "))
		fmt.Fprintln(fs.Output(), "  (no flavor = generic / universal skill)")
		fs.PrintDefaults()
	}
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if flavor == "" && fs.NArg() > 0 {
		flavor = fs.Arg(0)
	}

	if !skillFlavors[flavor] {
		return fmt.Errorf("unknown flavor %q (supported: %s)", flavor, strings.Join(flavorNames(), ", "))
	}

	content := buildSkill(flavor)

	if *toOut {
		_, err := os.Stdout.WriteString(content)
		return err
	}

	if !*force {
		if _, err := os.Stat(*out); err == nil {
			return fmt.Errorf("%s already exists (use --force to overwrite or --stdout to print)", *out)
		}
	}
	if err := os.WriteFile(*out, []byte(content), 0o644); err != nil {
		return err
	}

	label := flavor
	if label == "" {
		label = "generic"
	}
	fmt.Printf("wrote %s (%s flavor, %d bytes)\n", *out, label, len(content))
	return nil
}

// flavorNames returns the explicit, sorted platform flavor names. The default
// flavor (no argument / "generic") is omitted since it is the implicit default.
func flavorNames() []string {
	names := make([]string, 0, len(skillFlavors))
	for k := range skillFlavors {
		if k == "" || k == "generic" {
			continue
		}
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// buildSkill renders the confluence-skill markdown for the given flavor.
// The platform-specific preamble is prepended to a shared command reference.
func buildSkill(flavor string) string {
	var preamble string
	switch flavor {
	case "", "generic":
		preamble = genericPreamble
	case "claude":
		preamble = claudePreamble
	case "codex":
		preamble = codexPreamble
	case "gemini":
		preamble = geminiPreamble
	case "opencode":
		preamble = opencodePreamble
	}
	return preamble + skillReference
}

const genericPreamble = `# Confluence CLI Skill

> Universal skill describing how an AI agent should use the ` + "`confluence-cli`" + ` tool.
> Place or import this file wherever your agent reads project instructions.

`

const claudePreamble = `---
name: confluence-cli
description: Read and write pages on a self-hosted Confluence (Server/Data Center) instance via the confluence-cli command-line tool. Use when the user wants to search, read, create, update, or comment on internal Confluence or wiki pages.
---

# Confluence CLI Skill

> **Install (Claude):** rename this file to ` + "`SKILL.md`" + ` and place it under a folder
> matching the skill name, e.g. ` + "`.claude/skills/confluence-cli/SKILL.md`" + `
> (the folder name must match the ` + "`name`" + ` field above).

`

const codexPreamble = `# Confluence CLI — Agent Instructions

> **Install (Codex):** append this content to your project ` + "`AGENTS.md`" + ` (or
> ` + "`~/.codex/AGENTS.md`" + ` for global use). Codex reads AGENTS.md as standard Markdown.

`

const geminiPreamble = `# Confluence CLI — Context

> **Install (Gemini CLI):** add this content to ` + "`GEMINI.md`" + ` at your project root
> (or ` + "`~/.gemini/GEMINI.md`" + ` for global), or import it with ` + "`@confluence-skill.md`" + `.

`

const opencodePreamble = `# Confluence CLI — Agent Rules

> **Install (opencode):** opencode reads ` + "`AGENTS.md`" + `; append this content there,
> or reference this ` + "`confluence-skill.md`" + ` from it.

`

// skillReference is the shared, platform-agnostic body: what the tool is, how
// to authenticate, the command reference, and usage guidance.
const skillReference = `## What this is

` + "`confluence-cli`" + ` is a small, dependency-free CLI for reading and writing pages on a
**self-hosted Confluence Server/Data Center** instance. Prefer it over calling the REST
API directly: it handles authentication, retries on transient errors, CQL search, and the
Confluence XHTML "storage" body format.

## Setup (authentication)

Set these environment variables before invoking the CLI:

| Variable | Required | Purpose |
|---|---|---|
| ` + "`CONFLUENCE_BASE_URL`" + ` | yes | Site base URL, e.g. ` + "`https://confluence.example.com`" + ` |
| ` + "`CONFLUENCE_TOKEN`" + ` | yes* | Personal Access Token (Bearer). Preferred. |
| ` + "`CONFLUENCE_USER`" + ` / ` + "`CONFLUENCE_PASSWORD`" + ` | yes* | Basic-auth fallback when no token |
| ` + "`CONFLUENCE_SPACE`" + ` | no | Default space key for ` + "`search`" + `/` + "`create`" + ` (lets you omit ` + "`--space`" + `) |

\* Provide **either** a token **or** user + password.

## Commands

Every command prints **JSON by default** (ideal for parsing); add ` + "`--output text`" + ` for a
human-readable summary.

### search — find pages
` + "```bash" + `
confluence-cli search --text "release notes" --space INFRA --limit 10
confluence-cli search --cql 'type=page AND title ~ "design"'
# With CONFLUENCE_SPACE set, --space can be omitted:
confluence-cli search --text "release notes"
` + "```" + `

### get — read a page
` + "```bash" + `
confluence-cli get --id 123456 --output text   # human summary
confluence-cli get --id 123456 --body          # print only the storage-format body
` + "```" + `

### create — new page
` + "```bash" + `
confluence-cli create --space INFRA --title "New Page" --body "<p>Hello</p>"
confluence-cli create --title "From file" --body-file page.xhtml      # uses CONFLUENCE_SPACE
echo "<p>piped</p>" | confluence-cli create --title "Piped" --body-file -
` + "```" + `

### update — edit a page (version auto-increments)
` + "```bash" + `
confluence-cli update --id 123456 --body-file updated.xhtml
` + "```" + `

### comment — add a comment
` + "```bash" + `
confluence-cli comment --id 123456 --body "<p>LGTM</p>"
` + "```" + `

### spaces / tree / children — navigate the hierarchy
` + "```bash" + `
confluence-cli spaces --type global                  # list spaces
confluence-cli tree --space DOCS --depth 3            # page tree of a space
confluence-cli tree --id 123456                       # subtree under a page
confluence-cli children --id 123456                   # direct child pages
` + "```" + `

### move — re-parent / promote / demote a page
` + "```bash" + `
confluence-cli move --id 123456 --parent 654321       # demote under 654321
confluence-cli move --id 123456 --parent 111000       # promote to a grandparent
` + "```" + `
Note: on Server/DC this re-parents via ancestors; moving to the very top level
(no parent) is not supported via REST.

### labels — organize pages
` + "```bash" + `
confluence-cli labels --id 123456                     # list labels
confluence-cli labels --id 123456 --add "docs,runbook" --remove "draft"
` + "```" + `

### delete — remove a page
` + "```bash" + `
confluence-cli delete --id 123456 --yes               # skip the confirmation prompt
` + "```" + `

## Body format (important)

Confluence page bodies are **not Markdown**. They use Confluence's XHTML **storage format**.

- Default ` + "`--representation storage`" + ` expects storage-format XHTML:
  ` + "`<p>…</p>`" + `, ` + "`<h1>…</h1>`" + `, ` + "`<ac:structured-macro …/>`" + ` for macros (code blocks, panels, etc.).
- Use ` + "`--representation wiki`" + ` to send Confluence **wiki markup**; the server converts it.
- Do **not** pass raw Markdown — it would be stored literally.

## Usage guidance for agents

- Use ` + "`--output json`" + ` when you need to parse results; ` + "`--output text`" + ` for quick checks.
- ` + "`update`" + ` fetches the current version and increments it automatically — just pass the new body.
- Before editing, read the current body with ` + "`confluence-cli get --id <ID> --body`" + `.
- Returned objects include an ` + "`id`" + ` and ` + "`_links`" + `; surface the page URL to the user when done.
- Never hardcode credentials; rely on the environment variables above.
- Confirm destructive or outward-facing writes (create/update/comment on shared spaces) with the user first.
`
