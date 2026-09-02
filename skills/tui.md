# Skill: TUI with Bubbletea + Lipgloss

## Bubbletea Model

```go
type model struct {
    branches []Branch
    cursor   int
    selected map[int]struct{}
}
```

- `Init()` → return nil (no initial command)
- `Update(msg)` → handle key events (up/down/space/a/i/enter/q)
- `View()` → render with lipgloss styles

## Key Bindings

| Key | Action |
|---|---|
| `↑/↓` or `j/k` | Navigate |
| `space` | Toggle selection |
| `a` | Select all |
| `i` | Invert selection |
| `enter` | Confirm & delete |
| `q` / `ctrl+c` | Quit |

## Lipgloss Styles

- Color-code branch status: merged=green, squash-merged=cyan, gone=yellow, protected=red, active=white
- Table layout for branch list with columns: status, name, last commit date
