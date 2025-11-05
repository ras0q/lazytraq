package root

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ras0q/lazytraq/internal/traqapi"
	"github.com/ras0q/lazytraq/internal/traqapiext"
	"github.com/ras0q/lazytraq/internal/tui/shared"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/content"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/sidebar"
)

type rootModel struct {
	sidebar tea.Model
	content *content.MainViewModel
	ErrCh   chan error

	focus focusArea
}

type focusArea int

const (
	focusAreaSidebar focusArea = iota
	focusAreaContent
)

func New(w, h int) (*rootModel, error) {
	traqClient, err := traqapi.NewClient(
		"https://q.trap.jp/api/v3",
		traqapiext.NewSecuritySource(),
	)
	if err != nil {
		return nil, fmt.Errorf("create traq client: %w", err)
	}

	sidebarWidth := int(float64(w) * 0.3)
	contentWidth := w - sidebarWidth

	return &rootModel{
		sidebar: sidebar.New(sidebarWidth, h, traqClient),
		content: content.New(contentWidth, h, traqClient),
		ErrCh:   make(chan error),
	}, nil
}

var _ tea.Model = (*rootModel)(nil)

func (m *rootModel) Init() tea.Cmd {
	return tea.Batch(
		m.sidebar.Init(),
		m.content.Init(),
	)
}

func (m *rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0, 10)
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case error:
		m.ErrCh <- msg
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case shared.ReturnToSidebarMsg:
		m.focus = focusAreaSidebar

	case shared.OpenChannelMsg:
		channelID := msg.ID
		m.focus = focusAreaContent
		cmd = m.content.GetMessagesCmd(context.Background(), channelID)
		cmds = append(cmds, cmd)
	}

	switch m.focus {
	case focusAreaSidebar:
		m.sidebar, cmd = m.sidebar.Update(msg)
		cmds = append(cmds, cmd)

	case focusAreaContent:
		_content, cmd := m.content.Update(msg)
		m.content = _content.(*content.MainViewModel)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *rootModel) View() string {
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		withBorder(m.sidebar.View(), m.focus == focusAreaSidebar),
		withBorder(m.content.View(), m.focus == focusAreaContent),
	)
}

var borderStyle = lipgloss.NewStyle().
	Border(lipgloss.DoubleBorder())

var focusedBorderStyle = borderStyle.
	BorderForeground(lipgloss.Color("205"))

func withBorder(s string, focused bool) string {
	if focused {
		return focusedBorderStyle.Render(s)
	}

	return borderStyle.Render(s)
}
