# AGENTS.md

> Configuration for AI coding agents working on this repository.

## Project

**git-purge** — A CLI/TUI tool in Go that safely cleans up dead local Git branches (merged, squash-merged, or whose remote was deleted), without ever deleting unmerged work.

## Quick Commands

```bash
go build ./...                    # Build
go run ./cmd/git-purge            # Run locally
go test ./...                     # Unit tests
go test ./... -run Integration -v # Integration tests (temp git repos)
golangci-lint run                 # Lint
```

Always run `go test ./...` and `golangci-lint run` before considering a task complete.

## Rules

See `rules/` directory:

- `rules/safety.md` — Non-negotiable safety rules (data loss prevention)
- `rules/conventions.md` — Code style, testing patterns, workflow

## Skills

See `skills/` directory:

- `skills/go.md` — Go development patterns
- `skills/git-internals.md` — Git plumbing commands & squash-merge detection
- `skills/tui.md` — Bubbletea + Lipgloss TUI
- `skills/cli.md` — Cobra CLI structure
- `skills/ci.md` — GitHub Actions & goreleaser

## Plan

See `PLAN.md` for phased development plan. Work phase by phase, one commit = one phase.
