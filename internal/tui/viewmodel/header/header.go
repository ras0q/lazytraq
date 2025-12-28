package header

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	w, h    int
	apiHost string
}

func New(w, h int, apiHost string) *Model {
	return &Model{
		w:       w,
		h:       h,
		apiHost: apiHost,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

var titleStyle = lipgloss.NewStyle().
	Bold(true).
	Italic(true)

func (m *Model) View() string {
	return lipgloss.NewStyle().Height(m.h).Width(m.w).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Center,
			titleStyle.Render("lazytraq"),
			" in ",
			lipgloss.NewStyle().Bold(true).Render(m.apiHost),
		),
	)
}
