package root

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ras0q/lazytraq/internal/traqapi"
	"github.com/ras0q/lazytraq/internal/traqapiext"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/content"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/sidebar"
)

type rootModel struct {
	sidebar tea.Model
	content tea.Model
	errCh   chan error
}

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
		errCh:   make(chan error),
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
	f, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer func() { _ = f.Close() }()
	_, _ = fmt.Fprintf(f, "update message: %v\n", msg)

	switch msg := msg.(type) {
	case error:
		m.errCh <- msg
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	cmds := make([]tea.Cmd, 0, 10)
	var cmd tea.Cmd

	m.sidebar, cmd = m.sidebar.Update(msg)
	cmds = append(cmds, cmd)

	m.content, cmd = m.content.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *rootModel) View() string {
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.sidebar.View(),
		m.content.View(),
	)
}
