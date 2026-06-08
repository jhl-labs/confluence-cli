# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-06-08

### Added
- `spaces` — list spaces (`--type global|personal`).
- `tree` — render a space's page tree or a subtree (`--space` / `--id`, `--depth`).
- `children` — list a page's direct child pages.
- `move` — re-parent / promote / demote a page. Uses the Server/DC-compatible
  ancestors-update approach.
- `labels` — list, add, and remove labels on a page.
- `delete` — delete a page (confirmation prompt; `--yes` to skip).
- CI, Release, and CodeQL GitHub Actions workflows.
- Issue templates (bug, feature, commercial/usage permission), PR template,
  Dependabot config, and `SECURITY.md`.

### Changed
- License changed from MIT to the **JHL License** (non-commercial; commercial
  use of both binary and source requires the Licensor's written permission).
- `generate-skill` output now documents all commands.

### Notes
- Moving a page to the very top level (no parent) is not supported via the REST
  API on Confluence Server/Data Center; use the Confluence UI for that.

## [0.1.0] - 2026-06-08

### Added
- Initial release.
- Commands: `search` (CQL/text), `get`, `create`, `update`, `comment`.
- `generate-skill` — emit an agent skill doc for claude / codex / gemini /
  opencode / generic.
- Authentication via Personal Access Token (Bearer) or Basic; configuration
  from flags, environment variables, or a config file.
- `CONFLUENCE_SPACE` default space for `search` / `create`.
- Resilient HTTP client: retries on 429/5xx honoring `Retry-After`.
- JSON-first output with `--output text` for human-readable summaries.
- Cross-platform single binary; GitHub Pages landing page and install script.

[Unreleased]: https://github.com/jhl-labs/confluence-cli/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/jhl-labs/confluence-cli/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/jhl-labs/confluence-cli/releases/tag/v0.1.0
