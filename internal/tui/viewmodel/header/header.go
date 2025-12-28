package header

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	w, h int
}

func New(w, h int) *Model {
	return &Model{
		w: w,
		h: h,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *Model) View() string {
	return lipgloss.NewStyle().
		Width(m.w).
		Height(m.h).
		Bold(true).
		Italic(true).
		Render("lazytraq")
}
