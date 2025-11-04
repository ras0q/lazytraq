package sidebar

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/ras0q/lazytraq/internal/traqapi"
)

type SidebarModel struct {
	w, h             int
	channelTreeModel tea.Model
}

var _ tea.Model = (*SidebarModel)(nil)

func New(w, h int, traqClient *traqapi.Client) *SidebarModel {
	return &SidebarModel{
		w:                w,
		h:                h,
		channelTreeModel: NewChannelsModel(w-2, int(float64(h)*0.3)-2, traqClient),
	}
}

func (m *SidebarModel) Init() tea.Cmd {
	return tea.Batch(
		m.channelTreeModel.Init(),
	)
}

func (m *SidebarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0, 2)
	var cmd tea.Cmd

	m.channelTreeModel, cmd = m.channelTreeModel.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *SidebarModel) View() string {
	return lipgloss.NewStyle().
		Width(m.w).
		Height(m.h).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				lipgloss.NewStyle().
					Border(lipgloss.DoubleBorder()).
					Render(m.channelTreeModel.View()),
				lipgloss.NewStyle().
					Border(lipgloss.DoubleBorder()).
					Render(m.channelTreeModel.View()),
				lipgloss.NewStyle().
					Border(lipgloss.DoubleBorder()).
					Render(m.channelTreeModel.View()),
			),
		)
}
