# Security

## Reporting a vulnerability

Please report security issues privately via [GitHub Security Advisories](https://github.com/kioie/memex/security/advisories/new) rather than in public issues.

We will acknowledge reports promptly and work on fixes as needed.

## Scope

memex is a **local-first** MCP server. By default it:

- Runs over **stdio** (no network listener)
- Stores data in a **local SQLite file** (`~/.memex/memex.db` by default)
- Does not send memory content to external services

Review any custom deployments that expose memex over HTTP or shared filesystems.

## Automated security checks

CI runs:

- **CodeQL** static analysis (`.github/workflows/codeql.yml`)
- **govulncheck** for known Go module vulnerabilities (`.github/workflows/security.yml`)
- **SonarQube** static analysis with an **80% coverage floor** and strict quality gate enforcement (`.github/workflows/sonar.yml`; requires `SONAR_TOKEN`)
- **Dependabot** for dependency and GitHub Actions updates

## Threat notes

- Memory **content is stored as plain text** in SQLite unless you add encryption at the deployment layer.
- FTS queries are **sanitized** (token quoting) to reduce query injection risk; treat recalled memory as untrusted input in agent prompts.
- The MCP server inherits the **privileges of the host process** that launches it (Cursor, Claude Desktop, etc.).
