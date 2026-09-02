# Skill: Git Internals

## Key Plumbing Commands

| Command | Purpose |
|---|---|
| `git for-each-ref --format=... refs/heads/` | List local branches with metadata |
| `git branch -vv` | Branches with upstream tracking info |
| `git merge-base <a> <b>` | Find common ancestor |
| `git diff-tree -r <commit>` | List all files changed in a commit |
| `git branch --merged` | List branches fully merged (fast-forward) |
| `git reflog` | Recovery log for deleted branches |

## Squash-Merge Detection Algorithm

1. `merge-base(main, branch)` → find divergence point
2. `diff-tree` of branch vs merge-base → get the patch
3. Compare that diff (or resulting file state) against `main` history after merge-base
4. If identical → branch is `SQUASH_MERGED`

## Tracking Gone Branches

- `git branch -vv` output shows `[gone]` when remote tracking branch was deleted.
- Parse this output to identify candidates.
