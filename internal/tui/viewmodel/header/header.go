package header

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ras0q/lazytraq/internal/traqapi"
	"github.com/ras0q/lazytraq/internal/tui/shared"
)

type (
	meFetchedMsg *traqapi.MyUserDetail
)

type Model struct {
	w, h       int
	apiHost    string
	traqClient *traqapi.Client

	me *traqapi.MyUserDetail
}

func New(w, h int, apiHost string, traqClient *traqapi.Client) *Model {
	return &Model{
		w:          w,
		h:          h,
		apiHost:    apiHost,
		traqClient: traqClient,
	}
}

func (m *Model) Init() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		me, err := m.traqClient.GetMe(ctx)
		if err != nil {
			return shared.ErrorMsg(fmt.Errorf("fetch me: %w", err))
		}

		return meFetchedMsg(me)
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case meFetchedMsg:
		m.me = msg
	}

	return m, nil
}

var titleStyle = lipgloss.NewStyle().
	Bold(true).
	Italic(true)

func (m *Model) View() string {
	leftPart := lipgloss.JoinHorizontal(
		lipgloss.Center,
		titleStyle.Render("lazytraq"),
		" in ",
		lipgloss.NewStyle().Bold(true).Render(m.apiHost),
	)

	username := "@uknown"
	if m.me != nil {
		username = fmt.Sprintf("@%s", m.me.Name)
	}
	rightPart := lipgloss.NewStyle().
		Bold(true).
		Width(m.w - lipgloss.Width(leftPart) - 1).
		Align(lipgloss.Right).
		Render(username)

	return lipgloss.NewStyle().Height(m.h).Width(m.w).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			leftPart,
			rightPart,
		),
	)
}
