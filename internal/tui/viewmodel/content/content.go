package content

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/google/uuid"
	"github.com/ras0q/lazytraq/internal/traqapi"
	"github.com/ras0q/lazytraq/internal/traqapiext"
	"github.com/ras0q/lazytraq/internal/tui/shared"
)

type Model struct {
	w, h              int
	traqClient        *traqapi.Client
	messagesListModel list.Model
}

var _ tea.Model = (*Model)(nil)

func New(w, h int, traqClient *traqapi.Client) *Model {
	messagesListModel := list.New(
		[]list.Item{},
		list.NewDefaultDelegate(),
		w,
		h,
	)
	messagesListModel.DisableQuitKeybindings()

	return &Model{
		w:                 w,
		h:                 h,
		traqClient:        traqClient,
		messagesListModel: messagesListModel,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	ctx := context.Background()
	cmds := make([]tea.Cmd, 0, 10)

	switch msg := msg.(type) {
	case *traqapi.GetMessagesOKHeaders:
		messages := msg.Response
		slices.Reverse(messages)

		items := make([]list.Item, 0, len(messages))
		for _, message := range messages {
			items = append(items, traqapiext.MessageItem{
				Message: message,
			})
		}

		// TODO: cache previous items
		m.messagesListModel.SetItems(items)
		m.messagesListModel.Select(len(messages) - 1)

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			cmds = append(cmds, func() tea.Msg {
				return shared.ReturnToSidebarMsg{}
			})

		case "r":
			cmds = append(cmds, m.GetMessagesCmd(ctx, uuid.MustParse("f58c72a4-14f0-423c-9259-dbb4a90ca35f")))
		}
	}

	var cmd tea.Cmd

	m.messagesListModel, cmd = m.messagesListModel.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	return lipgloss.NewStyle().
		Width(m.w).
		Height(m.h).
		Render(m.messagesListModel.View())
}

func (m *Model) GetMessagesCmd(ctx context.Context, channelID uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		res, err := m.traqClient.GetMessages(ctx, traqapi.GetMessagesParams{
			ChannelId: channelID,
		})
		if err != nil {
			return fmt.Errorf("get messages from traQ: %w", err)
		}

		switch res := res.(type) {
		case *traqapi.GetMessagesOKHeaders:
			return res

		case *traqapi.GetMessagesBadRequest:
			return errors.New("bad request")

		case *traqapi.GetMessagesNotFound:
			return errors.New("not found")

		default:
			return fmt.Errorf("unreachable error")
		}
	}
}
