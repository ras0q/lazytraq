package content

import (
	"context"
	"fmt"
	"slices"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/ras0q/lazytraq/internal/traqapi"
	"github.com/ras0q/lazytraq/internal/traqapiext"
	"github.com/ras0q/lazytraq/internal/tui/shared"
)

type (
	messagesFetchedMsg []traqapi.Message
	usersFetchedMsg    map[uuid.UUID]traqapi.User
	stampsFetchedMsg   map[uuid.UUID]traqapi.StampWithThumbnail
)

type State struct {
	users  map[uuid.UUID]traqapi.User
	stamps map[uuid.UUID]traqapi.StampWithThumbnail
}

type Model struct {
	w, h              int
	traqContext       *traqapiext.Context
	messagesListModel list.Model

	state State
}

var _ tea.Model = (*Model)(nil)

func New(w, h int, traqContext *traqapiext.Context) *Model {
	messagesListModel := list.New(
		[]list.Item{},
		newListDelegate(),
		w,
		h,
	)
	messagesListModel.DisableQuitKeybindings()
	messagesListModel.SetShowTitle(false)
	messagesListModel.SetShowHelp(false)
	messagesListModel.SetShowPagination(false)

	return &Model{
		w:                 w,
		h:                 h,
		traqContext:       traqContext,
		messagesListModel: messagesListModel,
	}
}

func (m *Model) Init() tea.Cmd {
	ctx := context.Background()

	return tea.Batch(
		m.fetchUsersCmd(ctx),
		m.fetchStampsCmd(ctx),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	ctx := context.Background()
	cmds := make([]tea.Cmd, 0, 10)

	switch msg := msg.(type) {
	case messagesFetchedMsg:
		messages := msg
		slices.Reverse(messages)

		items := make([]list.Item, 0, len(messages))
		for _, message := range messages {
			user := m.state.users[message.GetUserId()]
			items = append(items, traqapiext.MessageItem{
				Message: message,
				User:    user,
			})
		}

		// TODO: cache previous items
		m.messagesListModel.SetItems(items)
		m.messagesListModel.Select(len(messages) - 1)

	case usersFetchedMsg:
		m.state.users = msg

	case stampsFetchedMsg:
		m.state.stamps = msg

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			cmds = append(cmds, func() tea.Msg {
				return shared.ReturnToSidebarMsg{}
			})

		case "r":
			cmds = append(cmds, m.FetchMessagesCmd(ctx, uuid.MustParse("f58c72a4-14f0-423c-9259-dbb4a90ca35f")))
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

func (m *Model) FetchMessagesCmd(ctx context.Context, channelID uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		messages, err := m.traqContext.Messages.Get(ctx, channelID)
		if err != nil {
			return shared.ErrorMsg(fmt.Errorf("get messages from traQ: %w", err))
		}

		return messagesFetchedMsg(messages)
	}
}

func (m *Model) fetchUsersCmd(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		users, err := m.traqContext.Users.Get(ctx, struct{}{})
		if err != nil {
			return shared.ErrorMsg(fmt.Errorf("get users from traQ: %w", err))
		}

		userMap := make(map[uuid.UUID]traqapi.User)
		for _, user := range users {
			userMap[user.ID] = user
		}

		return usersFetchedMsg(userMap)
	}
}

func (m *Model) fetchStampsCmd(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		stamps, err := m.traqContext.Stamps.Get(ctx, struct{}{})
		if err != nil {
			return shared.ErrorMsg(fmt.Errorf("get stamps from traQ: %w", err))
		}

		stampMap := make(map[uuid.UUID]traqapi.StampWithThumbnail)
		for _, stamp := range stamps {
			stampMap[stamp.ID] = stamp
		}

		return stampsFetchedMsg(stampMap)
	}
}
