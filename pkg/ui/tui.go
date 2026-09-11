package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jbdanho/git-purge/pkg/analyzer"
	"github.com/jbdanho/git-purge/pkg/safety"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).PaddingBottom(1)
	statusStyle   = lipgloss.NewStyle().Width(15)
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	helpStyle     = lipgloss.NewStyle().Faint(true).MarginTop(1)
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
)

func statusColor(s analyzer.Status) lipgloss.Style {
	switch s {
	case analyzer.StatusMerged:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	case analyzer.StatusSquashMerged:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("6")) // cyan
	case analyzer.StatusGone:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	case analyzer.StatusProtected:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("7")) // white
	}
}

// Model is the interactive branch selection UI.
type Model struct {
	candidates []analyzer.BranchResult
	cursor     int
	selected   map[int]bool
	engine     *safety.Engine
	done       bool
	deleted    int
	kept       int
	quitting   bool
	confirming bool
}

// New creates the TUI model.
func New(engine *safety.Engine, candidates []analyzer.BranchResult) *Model {
	return &Model{
		candidates: candidates,
		selected:   make(map[int]bool),
		engine:     engine,
	}
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	return nil
}

type deletionResult struct {
	name  string
	error error
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirming {
			return m.updateConfirm(msg)
		}
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.candidates)-1 {
				m.cursor++
			}
		case " ":
			if len(m.candidates) > 0 {
				m.selected[m.cursor] = !m.selected[m.cursor]
			}
		case "a":
			m.selectAll()
		case "i":
			m.invert()
		case "enter":
			if m.hasSelection() {
				m.confirming = true
			}
		}
	case deletionResult:
		if msg.error != nil {
			m.kept++
		} else {
			m.deleted++
		}
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "y":
		return m.deleteSelected()
	case "n", "q", "esc":
		m.confirming = false
		return m, nil
	}
	return m, nil
}

func (m *Model) selectAll() {
	for i := range m.candidates {
		m.selected[i] = true
	}
}

func (m *Model) invert() {
	for i := range m.candidates {
		m.selected[i] = !m.selected[i]
	}
}

func (m *Model) hasSelection() bool {
	for _, v := range m.selected {
		if v {
			return true
		}
	}
	return false
}

func (m *Model) deleteSelected() (tea.Model, tea.Cmd) {
	for i, c := range m.candidates {
		if !m.selected[i] {
			continue
		}
		if err := m.engine.DeleteSafe(c.Branch, 0, true); err != nil {
			return m, func() tea.Msg { return deletionResult{name: c.Name, error: err} }
		}
	}
	return m, func() tea.Msg { return deletionResult{} }
}

// View implements tea.Model.
func (m *Model) View() string {
	if m.quitting || m.done {
		if m.deleted > 0 || m.kept > 0 {
			var b strings.Builder
			b.WriteString(successStyle.Render(fmt.Sprintf("\n%d branch(es) deleted.\n", m.deleted)))
			if m.kept > 0 {
				b.WriteString(fmt.Sprintf("%d branch(es) could not be deleted.\n", m.kept))
			}
			b.WriteString("Recovery: check .git-purge/ logs or use git reflog.\n")
			return b.String()
		}
		return "\nExited. Nothing changed.\n"
	}

	if m.confirming {
		return m.confirmationView()
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("git-purge — pick branches to delete"))
	if len(m.candidates) == 0 {
		b.WriteString("No stale branches to clean. All clean!\n")
		return b.String()
	}

	for i, c := range m.candidates {
		if i == m.cursor {
			b.WriteString(cursorStyle.Render("> "))
		} else {
			b.WriteString("  ")
		}
		marker := " "
		if m.selected[i] {
			marker = "x"
		}
		sel := selectedStyle.Render(fmt.Sprintf("[%s]", marker))
		st := statusColor(c.Status).Render(statusStyle.Render(c.Status.String()))
		cur := ""
		if c.IsCurrent {
			cur = " (current)"
		}
		b.WriteString(fmt.Sprintf("%s %s %s%s\n", sel, st, c.Name, cur))
	}

	b.WriteString(helpStyle.Render("[space] toggle  [a] all  [i] invert  [enter] delete  [q] quit"))
	return b.String()
}

func (m *Model) confirmationView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Confirm deletion"))
	for i, c := range m.candidates {
		if m.selected[i] {
			st := statusColor(c.Status).Render(c.Status.String())
			b.WriteString(fmt.Sprintf("  x %s %s\n", st, c.Name))
		}
	}
	b.WriteString(successStyle.Render("\n[y] yes") + "   " + errorStyle.Render("[n] no"))
	return b.String()
}
