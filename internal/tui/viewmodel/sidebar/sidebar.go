package sidebar

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/ras0q/lazytraq/internal/traqapi"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/sidebar/channeltree"
)

type Model struct {
	w, h             int
	channelTreeModel *channeltree.Model
}

var _ tea.Model = (*Model)(nil)

func New(w, h int, traqClient *traqapi.Client) *Model {
	return &Model{
		w:                w,
		h:                h,
		channelTreeModel: channeltree.New(w-2, h-2, traqClient),
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
