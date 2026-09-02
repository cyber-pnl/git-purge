# Skill: Go Development

## Patterns

- Use `exec.Command("git", ...)` for Git operations — more reliable than `go-git` for edge cases like `diff-tree`.
- Errors: always wrap with `fmt.Errorf("context: %w", err)`.
- No global state; pass dependencies via struct fields or function parameters.

## Testing

- Table-driven tests with `t.Run()` subtests.
- Real Git repos in temp dirs:
  ```go
  dir := t.TempDir()
  exec.Command("git", "init", dir).Run()
  ```
- Use `t.Helper()` in test helpers.

## Dependencies

- `github.com/spf13/cobra` — CLI framework
- `github.com/charmbracelet/bubbletea` — TUI
- `github.com/charmbracelet/lipgloss` — TUI styling
