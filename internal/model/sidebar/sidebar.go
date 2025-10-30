package sidebar

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/google/uuid"
	"github.com/ras0q/lazytraq/internal/traqapi"
	"github.com/ras0q/lazytraq/internal/traqapiext"
)

type SidebarModel struct {
	w, h              int
	spinnerModel      spinner.Model
	channelsListModel list.Model
}

var _ tea.Model = (*SidebarModel)(nil)

func New(w, h int) *SidebarModel {
	s := spinner.New(
		spinner.WithSpinner(spinner.Dot),
	)

	channelsListModel := list.New(
		[]list.Item{
			traqapiext.MessageItem{
				Message: traqapi.Message{
					UserId:  uuid.New(),
					Content: "Hello",
				},
			},
		},
		list.NewDefaultDelegate(),
		w,
		h-1,
	)

	return &SidebarModel{
		w:                 w,
		h:                 h,
		spinnerModel:      s,
		channelsListModel: channelsListModel,
	}
}

func (m *SidebarModel) Init() tea.Cmd {
	return m.spinnerModel.Tick
}

func (m *SidebarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	m.spinnerModel, cmd = m.spinnerModel.Update(msg)
	return m, cmd
}

func (m *SidebarModel) View() string {
	rootStyle := lipgloss.NewStyle().
		Width(m.w)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		rootStyle.Height(m.h).Render(m.spinnerModel.View()),
	)
}
