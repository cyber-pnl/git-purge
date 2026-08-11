
# Git-Purge

> The zero-friction local Git branch cleaner for modern developer workflows.

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=for-the-badge)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen?style=for-the-badge)]()

---

## The Pain Point

When Pull Requests are merged on GitHub, GitLab, or Bitbucket, remote branches are deleted automatically. However, your local machine accumulates dozens or hundreds of dead branches.

Safely clearing them out without losing unmerged work or active feature branches is a stressful, manual chore:
- `git branch -d` fails on Squash-Merged PRs because the commit hashes differ.
- `git branch -D` is dangerous and can cause accidental data loss.
- Manually checking branch status and running `git diff-tree` takes away valuable focus time.

`git-purge` solves this effortlessly.

---

## Features

- Smart Squash-Merge Detection: Safely identifies branches merged via "Squash & Merge" even when `git branch --merged` fails.
- Gone Branch Cleanup: Instantly detects branches whose remote counterparts have been pruned (`[gone]`).
- Interactive Terminal UI (TUI): Built with Go and Bubbletea for a smooth, keyboard-driven navigation experience.
- Fail-Safe & Non-Destructive:
  - Prevents accidental deletion of unmerged work or protected branches (`main`, `master`, `dev`, etc.).
  - Always records commit SHAs for quick `git reflog` recovery.
  - Supports `--dry-run` to inspect actions before applying them.
- Blazing Fast: Written in Go with minimal overhead — scans large repositories in milliseconds.

---

## Architecture Overview

`git-purge` is designed with a clear separation of concerns to guarantee performance and safety:

```text
┌───────────────────────────────────────────────────────────┐
│                    1. TUI / CLI Layer                     │
│         (Bubbletea TUI, Lipgloss styling, Cobra CLI)      │
└─────────────────────────────┬─────────────────────────────┘
                              │
                              ▼
┌───────────────────────────────────────────────────────────┐
│                    2. Branch Analyzer                     │
│  (Classification: Merged, Squash-Merged, Gone, Protected) │
└─────────────────────────────┬─────────────────────────────┘
                              │
                              ▼
┌───────────────────────────────────────────────────────────┐
│                    3. Git Engine Adapter                  │
│       (Executes git commands & tree diff operations)      │
└─────────────────────────────┬─────────────────────────────┘
                              │
                              ▼
┌───────────────────────────────────────────────────────────┐
│               4. Safety & Execution Engine                │
│    (Protected branch filter, Reflog logging, Safety checks)│
└─────────────────────────────┬─────────────────────────────┘

```

---

## Quick Start & Installation

### Prerequisite

Ensure Go 1.22+ is installed on your machine.

### Build & Install from Source

```bash
git clone [https://github.com/yourusername/git-purge.git](https://github.com/yourusername/git-purge.git)
cd git-purge
go install

```

### Binary Download (Release)

Download the latest pre-compiled binary for macOS, Linux, or Windows from the Releases page.

---

## Usage & CLI Commands

### 1. Interactive Mode (Default)

Launch the rich TUI to select, inspect, and safely purge branches interactively:

```bash
git-purge

```

### 2. Dry-Run Mode

Inspect what would be purged without actually deleting anything:

```bash
git-purge --dry-run

```

### 3. Non-Interactive / Automation Mode

Purge all safe candidates automatically (useful for cron jobs or CLI aliases):

```bash
git-purge --force --yes

```

### 4. Custom Safety Options

Protect specific branches or keep branches updated within a time window:

```bash
# Keep branches modified within the last 14 days
git-purge --keep-days 14

# Custom protected branch patterns
git-purge --protect "main,master,release/*,staging"

```

---

## How Squash-Merge Detection Works

When GitHub/GitLab performs a Squash and Merge, the commit hash on `main` does not match the commit hash on your local branch.

```text
Local Branch:  A ─── B ─── C  (Hash: 0x1234)
Main Branch:   ─── S (Squashed Commit, Hash: 0x9876)

```

Standard `git branch -d` relies on matching SHAs and will refuse to delete the branch. `git-purge` overcomes this safely:

1. Calculates the `merge-base` commit between `main` and your local branch.
2. Generates a tree comparison (`git diff-tree`) of the changes on your branch.
3. Checks if the identical patch or resulting file state exists within `main`.
4. If the diff matches, the branch is safely categorized as `SQUASH_MERGED` and can be removed without data loss.

---

## Project Structure

```text
git-purge/
├── cmd/
│   └── git-purge/          # Application entrypoint (main.go, Cobra commands)
├── pkg/
│   ├── analyzer/           # Branch classification algorithms & squash detection
│   ├── git/                # Git adapter (wraps exec.Command or go-git)
│   ├── safety/             # Protected branch rules, working-tree validation
│   └── ui/                 # Bubbletea TUI models, views, and Lipgloss styles
├── go.mod
├── go.sum
├── LICENSE
└── README.md

```

---

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

