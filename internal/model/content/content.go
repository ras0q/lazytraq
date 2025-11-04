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
)

type MainViewModel struct {
	w, h              int
	traqClient        *traqapi.Client
	messagesListModel list.Model
}

var _ tea.Model = (*MainViewModel)(nil)

func New(w, h int, traqClient *traqapi.Client) *MainViewModel {
	return &MainViewModel{
		w:          w,
		h:          h,
		traqClient: traqClient,
		messagesListModel: list.New(
			[]list.Item{},
			list.NewDefaultDelegate(),
			w,
			h,
		),
	}
}

func (m *MainViewModel) Init() tea.Cmd {
	return nil
}

func (m *MainViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	ctx := context.Background()
	cmds := make([]tea.Cmd, 0, 10)

	switch msg := msg.(type) {
	case *traqapi.GetMessagesOKHeaders:
		messages := msg.Response
		slices.Reverse(messages)

		itemsLen := len(m.messagesListModel.Items())
		for i, message := range messages {
			m.messagesListModel.InsertItem(itemsLen+i, traqapiext.MessageItem{
				Message: message,
			})
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			cmds = append(cmds, m.getMessagesCmd(ctx, uuid.MustParse("f58c72a4-14f0-423c-9259-dbb4a90ca35f")))
		}
	}

	var cmd tea.Cmd

	m.messagesListModel, cmd = m.messagesListModel.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *MainViewModel) View() string {
	return lipgloss.NewStyle().
		Width(m.w - 2).
		Height(m.h - 2).
		Border(lipgloss.DoubleBorder()).
		Render(m.messagesListModel.View())
}

func (m *MainViewModel) getMessagesCmd(ctx context.Context, channelID uuid.UUID) tea.Cmd {
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
