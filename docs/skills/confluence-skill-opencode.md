# Confluence CLI — Agent Rules

> **Install (opencode):** opencode reads `AGENTS.md`; append this content there,
> or reference this `confluence-skill.md` from it.

## What this is

`confluence-cli` is a small, dependency-free CLI for reading and writing pages on a
**self-hosted Confluence Server/Data Center** instance. Prefer it over calling the REST
API directly: it handles authentication, retries on transient errors, CQL search, and the
Confluence XHTML "storage" body format.

## Setup (authentication)

Set these environment variables before invoking the CLI:

| Variable | Required | Purpose |
|---|---|---|
| `CONFLUENCE_BASE_URL` | yes | Site base URL, e.g. `https://confluence.example.com` |
| `CONFLUENCE_TOKEN` | yes* | Personal Access Token (Bearer). Preferred. |
| `CONFLUENCE_USER` / `CONFLUENCE_PASSWORD` | yes* | Basic-auth fallback when no token |
| `CONFLUENCE_SPACE` | no | Default space key for `search`/`create` (lets you omit `--space`) |

\* Provide **either** a token **or** user + password.

## Commands

Every command prints **JSON by default** (ideal for parsing); add `--output text` for a
human-readable summary.

### search — find pages
```bash
confluence-cli search --text "release notes" --space INFRA --limit 10
confluence-cli search --cql 'type=page AND title ~ "design"'
# With CONFLUENCE_SPACE set, --space can be omitted:
confluence-cli search --text "release notes"
```

### get — read a page
```bash
confluence-cli get --id 123456 --output text   # human summary
confluence-cli get --id 123456 --body          # print only the storage-format body
```

### create — new page
```bash
confluence-cli create --space INFRA --title "New Page" --body "<p>Hello</p>"
confluence-cli create --title "From file" --body-file page.xhtml      # uses CONFLUENCE_SPACE
echo "<p>piped</p>" | confluence-cli create --title "Piped" --body-file -
```

### update — edit a page (version auto-increments)
```bash
confluence-cli update --id 123456 --body-file updated.xhtml
```

### comment — add a comment
```bash
confluence-cli comment --id 123456 --body "<p>LGTM</p>"
```

### spaces / tree / children — navigate the hierarchy
```bash
confluence-cli spaces --type global                  # list spaces
confluence-cli tree --space DOCS --depth 3            # page tree of a space
confluence-cli tree --id 123456                       # subtree under a page
confluence-cli children --id 123456                   # direct child pages
```

### move — re-parent / promote / demote a page
```bash
confluence-cli move --id 123456 --parent 654321       # demote under 654321
confluence-cli move --id 123456 --parent 111000       # promote to a grandparent
```
Note: on Server/DC this re-parents via ancestors; moving to the very top level
(no parent) is not supported via REST.

### labels — organize pages
```bash
confluence-cli labels --id 123456                     # list labels
confluence-cli labels --id 123456 --add "docs,runbook" --remove "draft"
```

### delete — remove a page
```bash
confluence-cli delete --id 123456 --yes               # skip the confirmation prompt
```

## Body format (important)

Confluence page bodies are **not Markdown**. They use Confluence's XHTML **storage format**.

- Default `--representation storage` expects storage-format XHTML:
  `<p>…</p>`, `<h1>…</h1>`, `<ac:structured-macro …/>` for macros (code blocks, panels, etc.).
- Use `--representation wiki` to send Confluence **wiki markup**; the server converts it.
- Do **not** pass raw Markdown — it would be stored literally.

## Usage guidance for agents

- Use `--output json` when you need to parse results; `--output text` for quick checks.
- `update` fetches the current version and increments it automatically — just pass the new body.
- Before editing, read the current body with `confluence-cli get --id <ID> --body`.
- Returned objects include an `id` and `_links`; surface the page URL to the user when done.
- Never hardcode credentials; rely on the environment variables above.
- Confirm destructive or outward-facing writes (create/update/comment on shared spaces) with the user first.
