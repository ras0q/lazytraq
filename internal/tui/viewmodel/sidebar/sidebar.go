package sidebar

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ras0q/lazytraq/internal/traqapiext"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/sidebar/channeltree"
)

type Model struct {
	w, h             int
	channelTreeModel *channeltree.Model
}

var _ tea.Model = (*Model)(nil)

func New(w, h int, traqContext *traqapiext.Context) *Model {
	return &Model{
		w:                w,
		h:                h,
		channelTreeModel: channeltree.New(w-2, h-2, traqContext),
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.channelTreeModel.Init(),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_channelTreeModel, cmd := m.channelTreeModel.Update(msg)
	m.channelTreeModel = _channelTreeModel.(*channeltree.Model)

	return m, cmd
}

func (m *Model) View() string {
	return lipgloss.NewStyle().
		Width(m.w).
		Height(m.h).
		Render(m.channelTreeModel.View())
}
