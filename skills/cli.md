# Skill: CLI with Cobra

## Command Structure

```
git-purge              # Default: interactive TUI
git-purge list         # List branches (classified, no action)
git-purge version      # Print version
```

## Persistent Flags (all commands)

| Flag | Type | Default | Description |
|---|---|---|---|
| `--dry-run` | bool | false | Simulate without deleting |
| `--protect` | string | "" | Comma-separated protected branch patterns |
| `--keep-days` | int | 0 | Exclude branches modified within N days |
| `--verbose` | bool | false | Verbose output |

## Local Flags

| Command | Flag | Type | Description |
|---|---|---|---|
| `root` | `--force` | bool | Skip confirmation (non-interactive) |
| `root` | `--yes` | bool | Auto-confirm all prompts |

## Exit Codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | General error |
| 2 | Not a git repo |
| 3 | Safety check failed |
