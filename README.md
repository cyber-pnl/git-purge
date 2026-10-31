# Git-Purge

> The zero-friction local Git branch cleaner for modern developer workflows.

[![CI](https://github.com/cyber-pnl/git-purge/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/cyber-pnl/git-purge/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/cyber-pnl/git-purge?style=for-the-badge)](https://github.com/cyber-pnl/git-purge/releases/latest)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=for-the-badge&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=for-the-badge)](LICENSE)

---

## The Problem

When PRs are merged on GitHub/GitLab/Bitbucket, remote branches get deleted automatically — but your local machine accumulates dozens of dead branches. Cleaning them safely without losing unmerged work is a manual, error-prone chore.

**git-purge** solves this effortlessly.

---

## Features

- **Smart Squash-Merge Detection** — identifies branches merged via squash even when `git branch --merged` fails
- **Gone Branch Cleanup** — detects branches whose remote was pruned
- **Optional Remote Cleanup** — also delete the matching remote branch, only once it is merged
- **Automatic `.gitignore` Hygiene** — the `.git-purge/` log directory is ignored in your project automatically
- **Interactive TUI** — keyboard-driven terminal UI built with Bubbletea
- **Fail-Safe & Non-Destructive** — protected branches, SHA logging, `--dry-run` support
- **Blazing Fast** — pure Go, scans large repos in milliseconds

---

## Installation

**Option 1 — Install system-wide (available for all users, no PATH setup needed):**

```bash
sudo env GOBIN=/usr/local/bin go install github.com/cyber-pnl/git-purge/cmd/git-purge@latest
```

The binary is placed in `/usr/local/bin`, which is always on the system `PATH`, so `git-purge` is immediately available in any terminal:

```bash
git-purge version   # => git-purge v0.2.0
```

**Option 2 — Install to your Go bin directory:**

```bash
go install github.com/cyber-pnl/git-purge/cmd/git-purge@latest

# make sure $(go env GOPATH)/bin is on your PATH
export PATH="$PATH:$(go env GOPATH)/bin"
```

**Option 3 — Download a pre-compiled binary** from the [Releases](https://github.com/cyber-pnl/git-purge/releases) page and add it to your `PATH`.

---

## Usage

```bash
# Interactive mode (default)
git-purge

# Preview what would be deleted
git-purge --dry-run

# Non-interactive purge (for scripts/cron)
git-purge --force --yes

# Keep branches modified in the last 14 days
git-purge --keep-days 14

# Protect custom branches
git-purge --protect "main,master,release/*,staging"

# Also delete the matching remote branch (merged branches only)
git-purge --delete-remote
```

---

## How It Works

**Branch classification** — every local branch is classified on a decision chain that never guesses: any doubt means the branch is kept.

```mermaid
flowchart TD
    B[branch] --> C{is current branch<br/>or main / master / dev?}
    C -- yes --> P[PROTECTED<br/>never deleted]
    C -- no --> G{upstream track<br/>= gone?}
    G -- yes --> GO[GONE<br/>remote pruned → safe to delete]
    G -- no --> M{fully merged into base?<br/>merge-base --is-ancestor}
    M -- yes --> ME[MERGED<br/>safe to delete]
    M -- no --> S{same tree on base<br/>since merge-base?}
    S -- yes --> SQ[SQUASH_MERGED<br/>squash-merge detected → safe to delete]
    S -- no --> A[ACTIVE_UNMERGED<br/>kept, never deleted]
```

**Squash-merge detection** — When a PR is squash-merged, the commit hash on `main` differs from the branch. git-purge compares the branch **tree** against `main`'s history to safely confirm the merge.

**Protected branches** — `main`, `master`, `dev`, and your current branch are never deleted. Custom patterns supported via `--protect`.

**Remote cleanup** — `--delete-remote` also removes the remote branch, but only when the local branch is `MERGED` or `SQUASH_MERGED` **and** the remote tip is exactly the same commit as the local tip. If the remote has extra commits, it is left untouched.

**Recovery log** — Every deletion records the branch SHA in `.git-purge/` for easy recovery via `git reflog` or `git branch <name> <sha>`. The log directory is added to the project's `.gitignore` automatically.

---

## Architecture

The codebase is layered so that safety decisions are enforced no matter which entry point you use:

```mermaid
flowchart TD
    CLI[internal/cli<br/>Cobra commands]
    CLI -->|list| LST[print classification]
    CLI -->|purge| RUN[runPurge]
    CLI -->|version| VER[print version]

    RUN --> DRY{dry-run?}
    DRY -- yes --> OUT[print candidates<br/>nothing deleted]
    DRY -- no --> FORCE{--force --yes?}
    FORCE -- yes --> DEL[deleteAll<br/>loop over candidates]
    FORCE -- no --> TUI[pkg/ui<br/>Bubbletea selection]

    TUI --> CONF[confirm selection]
    CONF --> DEL

    LST & RUN & DEL --> SAFETY[pkg/safety<br/>IsProtected / keep-days / CanDelete]
    SAFETY --> REC[record SHA in .git-purge<br/>+ ignore it in .gitignore]
    REC --> GIT[pkg/git<br/>git branch -d / -D<br/>git push --delete]
    GIT --> CLASS[pkg/analyzer<br/>branch classification]

    CLASS --> GIT
```

**Layer responsibilities**

```
internal/cli      → command handling, flags, orchestration (setup / purge / list)
pkg/analyzer      → decides WHAT a branch is (merged, squash-merged, gone, ...)
pkg/safety        → decides IF a branch may be deleted (protections, keep-days, logging)
pkg/git           → thin adapter over the `git` CLI (exec.Command)
pkg/ui            → interactive terminal UI (Bubbletea)
```

---

## Development

```bash
go build ./...                     # build the binary
go test ./...                      # unit tests
go test ./... -run Integration -v  # integration tests (real temp git repos)
golangci-lint run                  # lint
```

CI runs the unit tests on Go 1.25 and the latest stable release, the integration
suite, the linter, and the security scan below.

---

## Security

- **Runtime safety** — deletions are gated by `pkg/safety`, and every removed
  branch's SHA is recorded in `.git-purge/` so it can be restored with
  `git branch <name> <sha>`.
- **Continuous scanning** — every push and pull request runs
  [Trivy](https://trivy.dev) over dependencies, secrets and misconfigurations.
  Findings are uploaded as SARIF to the
  [Security tab](https://github.com/cyber-pnl/git-purge/security/code-scanning),
  and CI fails on any fixable `CRITICAL` or `HIGH` issue.

---

## License

[MIT](LICENSE)
