# Safety Rules — NON-NEGOTIABLE

These rules protect users from data loss. This project deletes Git branches locally — an error can destroy work.

## Core Rules

- **Never** run `git branch -D` without going through `pkg/safety` (protection filter + confirmation).
- Always record the branch SHA before deletion (log in `.git-purge/`), even in `--force --yes` mode.
- Branches `main`, `master`, `dev`, and the current branch (HEAD) are protected by default, with hardcoded exceptions that cannot be bypassed.
- `--dry-run` must produce exactly the same calculations as real mode — only the final execution (`DeleteBranch`) changes.
- A branch classified as `ACTIVE_UNMERGED` must never appear in the deletion candidates list, even in force mode.
- In case of doubt about a branch status (git error, ambiguity), classify it as **non-deletable** rather than the opposite.

## What the Agent Must Never Do Alone

- Modify or remove the safety rules in this file.
- Run destructive git commands directly in the development repo to "test" (use temporary repos only).
- Publish a release (`goreleaser release`) without explicit user confirmation.
