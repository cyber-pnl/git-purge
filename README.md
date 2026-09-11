# Git-Purge

> The zero-friction local Git branch cleaner for modern developer workflows.

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=for-the-badge)](LICENSE)

---

## The Problem

When PRs are merged on GitHub/GitLab/Bitbucket, remote branches get deleted automatically — but your local machine accumulates dozens of dead branches. Cleaning them safely without losing unmerged work is a manual, error-prone chore.

**git-purge** solves this effortlessly.

---

## Features

- **Smart Squash-Merge Detection** — identifies branches merged via squash even when `git branch --merged` fails
- **Gone Branch Cleanup** — detects branches whose remote was pruned
- **Interactive TUI** — keyboard-driven terminal UI built with Bubbletea
- **Fail-Safe & Non-Destructive** — protected branches, SHA logging, `--dry-run` support
- **Blazing Fast** — pure Go, scans large repos in milliseconds

---

## Installation

> [!NOTE]
> `go install ...@latest` and the Releases page only work once the repository is **published (public) on GitHub**.
> While the code is still local/unpublished, install directly from your working copy instead.

**From source (works before publishing):**

```bash
cd /path/to/git-purge
go build -o git-purge ./cmd/git-purge
sudo mv git-purge /usr/local/bin/   # optional: put on your PATH
```

**Once published on GitHub:**

```bash
go install github.com/jbdanho/git-purge/cmd/git-purge@latest
```

Or download a pre-compiled binary from the [Releases](https://github.com/jbdanho/git-purge/releases) page.

> If `go install` asks for a GitHub username/password, it means the repo is **private or not yet published** — the module must be public for `@latest` to resolve.

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
```

---

## How It Works

**Squash-merge detection** — When a PR is squash-merged, the commit hash on `main` differs from the branch. git-purge compares the branch diff against `main`'s history to safely confirm the merge.

**Protected branches** — `main`, `master`, `dev`, and your current branch are never deleted. Custom patterns supported via `--protect`.

**Recovery log** — Every deletion records the branch SHA in `.git-purge/` for easy recovery via `git reflog` or `git branch <name> <sha>`.

---

## Architecture

```
cmd/git-purge   →   pkg/ui (Bubbletea TUI)
                         ↓
                    pkg/analyzer (branch classification)
                         ↓
                    pkg/git (Git adapter via exec.Command)
                         ↓
                    pkg/safety (protections, dry-run, logging)
```

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[MIT](LICENSE)
