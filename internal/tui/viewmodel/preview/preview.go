package preview

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/ras0q/lazytraq/internal/traqapiext"
	"github.com/ras0q/lazytraq/internal/tui/shared"
)

type Model struct {
	w, h     int
	viewport viewport.Model
	renderer *glamour.TermRenderer

	message *traqapiext.MessageItem
}

var _ tea.Model = (*Model)(nil)

func New(w, h int) *Model {
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(w),
	)

	viewport := viewport.New(w, h)
	viewport.SetContent("No message selected.")

	return &Model{
		w:        w,
		h:        h - 1,
		viewport: viewport,
		renderer: renderer,
	}
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	switch msg := msg.(type) {
	case shared.PreviewMessageRenderedMsg:
		m.viewport.SetContent(string(msg))

	case shared.PreviewMessageMsg:
		m.message = msg

		cmd := func() tea.Msg {
			s, err := m.renderer.Render(m.message.Description())
			if err != nil {
				return shared.ErrorMsg(
					fmt.Errorf("render markdown: %w", err),
				)
			}

			return shared.PreviewMessageRenderedMsg(s)
		}
		cmds = append(cmds, cmd)

	case tea.KeyMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model.
func (m *Model) View() string {
	return m.viewport.View()
}
