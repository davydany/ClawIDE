# ClawIDE — Project Context

## What This Is
ClawIDE is a web-based IDE for managing multiple Claude Code sessions across projects. It is a single Go binary (`clawide`) that serves a web UI with tmux-backed terminal sessions.

## Project Structure
```
ccmux/
├── cmd/              # Go CLI entrypoints
├── internal/         # Go internal packages (server, tmux, version, etc.)
├── clawide/          # Go main package
├── web/              # Frontend assets (HTML, JS, CSS) served by Go binary
├── docs-website/     # Hugo static docs site (deployed separately to Heroku)
├── scripts/          # Build and utility scripts
├── .github/workflows/  # CI/CD (GitHub Actions)
├── Makefile          # Build commands for the Go binary
├── go.mod / go.sum   # Go dependencies
└── docker-compose.yml
```

## Main Application (Go)
- **Language**: Go 1.24
- **Build**: `make build` — produces `clawide` binary with version injection via ldflags
- **Frontend**: Vanilla JS + Tailwind CSS in `web/` — no React, no frameworks
- **Tests**: `make test` runs Go tests

## Docs Site (`docs-website/`)
- **Framework**: Hugo (v0.145.0) with Tailwind CSS v4
- **Build**: `npm run build` → `hugo --minify --gc`
- **Dev server**: `npm run dev` → `hugo server --buildDrafts`
- **Deployment**: GitHub Actions (`deploy-docs.yml`) does a `git subtree push` to Heroku app `clawide-docs`
- **Heroku app**: `clawide-docs` — config vars are available as env vars during Hugo build
- **Analytics**: Google Analytics injected via `HUGO_GOOGLE_ANALYTICS_ID` env var at build time (see `layouts/partials/analytics.html`)

## CI/CD
- **`deploy-docs.yml`**: Deploys `docs-website/` subtree to Heroku on push to `master` when `docs-website/**` changes
- **`ci.yml`**: CI checks
- **`release.yml`**: Release workflow
- **Secrets**: Heroku API key and other secrets are stored in GitHub Actions secrets and Heroku config vars — never hardcoded

## Self-Research Directive
Before asking the user questions about project structure, frameworks, tooling, or file locations — explore the codebase first. Use Glob, Grep, and Read to find answers. Only ask the user when the answer genuinely cannot be determined from the code.
