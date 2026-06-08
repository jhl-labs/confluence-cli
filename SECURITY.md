# Security Policy

## Reporting a vulnerability

Please **do not** open a public issue for security vulnerabilities.

Report privately via GitHub Security Advisories:
<https://github.com/jhl-labs/confluence-cli/security/advisories/new>

Include reproduction steps and affected versions where possible. When sharing
logs or configuration, **redact tokens, passwords, and internal hostnames**.

## Supported versions

The latest released version receives security fixes. Older versions are not
maintained.

## Handling secrets

confluence-cli authenticates with Confluence using a Personal Access Token or
Basic credentials. To keep these safe:

- Provide credentials via environment variables (`CONFLUENCE_TOKEN`,
  `CONFLUENCE_USER` / `CONFLUENCE_PASSWORD`) or a config file outside version
  control — never hardcode them.
- Keep config files (e.g. `~/.config/confluence-cli/config.json`) readable only
  by your user.
- Tokens are sent only to the configured `CONFLUENCE_BASE_URL` over HTTPS.
- Rotate any token that may have been exposed (logs, shell history, screen
  sharing) immediately.
