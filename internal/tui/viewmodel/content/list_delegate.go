package content

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/ras0q/lazytraq/internal/traqapiext"
	"github.com/ras0q/lazytraq/internal/tui/shared"
)

func newListDelegate() list.ItemDelegate {
	d := list.NewDefaultDelegate()

	var currentID uuid.UUID

	d.UpdateFunc = func(msg tea.Msg, m *list.Model) tea.Cmd {
		message, ok := m.SelectedItem().(traqapiext.MessageItem)
		if !ok {
			return nil
		}

		if currentID == message.Message.ID {
			return nil
		}

		currentID = message.Message.ID

		return func() tea.Msg {
			return shared.PreviewMessageMsg(&message)
		}
	}

	return d
}
