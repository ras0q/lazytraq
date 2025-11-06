package root

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/ras0q/lazytraq/internal/traqapi"
	"github.com/ras0q/lazytraq/internal/traqapiext"
	"github.com/ras0q/lazytraq/internal/tui/shared"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/content"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/sidebar"
)

type Model struct {
	sidebar *sidebar.Model
	content *content.Model
	ErrCh   chan error

	focus focusArea
}

type focusArea int

const (
	focusAreaSidebar focusArea = iota
	focusAreaContent
)

func New(w, h int) (*Model, error) {
	traqClient, err := traqapi.NewClient(
		"https://q.trap.jp/api/v3",
		traqapiext.NewSecuritySource(),
	)
	if err != nil {
		return nil, fmt.Errorf("create traq client: %w", err)
	}

	sidebarWidth := int(float64(w) * 0.3)
	contentWidth := w - sidebarWidth

	return &Model{
		sidebar: sidebar.New(sidebarWidth, h, traqClient),
		content: content.New(contentWidth, h, traqClient),
		ErrCh:   make(chan error),
	}, nil
}

var _ tea.Model = (*Model)(nil)

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.sidebar.Init(),
		m.content.Init(),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0, 10)

	switch msg := msg.(type) {
	case shared.ErrorMsg:
		m.ErrCh <- msg
		return m, tea.Quit

	case shared.ReturnToSidebarMsg:
		m.focus = focusAreaSidebar

	case shared.OpenChannelMsg:
		channelID := msg.ID
		if channelID == uuid.Nil {
			break
		}

		m.focus = focusAreaContent
		cmd := m.content.FetchMessagesCmd(context.Background(), channelID)
		cmds = append(cmds, cmd)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	switch m.focus {
	case focusAreaSidebar:
		_sidebar, cmd := m.sidebar.Update(msg)
		m.sidebar = _sidebar.(*sidebar.Model)
		cmds = append(cmds, cmd)

	case focusAreaContent:
		_content, cmd := m.content.Update(msg)
		m.content = _content.(*content.Model)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
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
