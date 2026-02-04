package timeline

import (
	"context"
	"fmt"
	"slices"

	"github.com/blacktop/go-termimg"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/ras0q/lazytraq/internal/traqapi"
	"github.com/ras0q/lazytraq/internal/traqapiext"
	"github.com/ras0q/lazytraq/internal/tui/shared"
)

type (
	messagesFetchedMsg []traqapi.Message
	usersFetchedMsg    map[uuid.UUID]traqapi.User
)

type State struct {
	messages []traqapi.Message
	users    map[uuid.UUID]traqapi.User
}

type Model struct {
	w, h        int
	traqContext *traqapiext.Context
	viewport    viewport.Model
	renderer    *glamour.TermRenderer
	theme       shared.Theme

	state State
}

var _ tea.Model = (*Model)(nil)

func New(w, h int, traqContext *traqapiext.Context, theme shared.Theme) *Model {
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(w-10),
	)

	vp := viewport.New(w, h)
	vp.SetContent("No messages yet.")

	return &Model{
		w:           w,
		h:           h,
		traqContext: traqContext,
		viewport:    vp,
		renderer:    renderer,
		theme:       theme,
	}
}

func (m *Model) Init() tea.Cmd {
	ctx := context.Background()

	return m.fetchUsersCmd(ctx)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0, 10)

	switch msg := msg.(type) {
	case messagesFetchedMsg:
		m.state.messages = msg
		slices.Reverse(m.state.messages)
		if err := m.renderTimeline(); err != nil {
			cmds = append(cmds, func() tea.Msg {
				return shared.ErrorMsg(fmt.Errorf("render timeline: %w", err))
			})
		}

	case usersFetchedMsg:
		m.state.users = msg

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			cmds = append(cmds, func() tea.Msg {
				return shared.ReturnToSidebarMsg{}
			})
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	return lipgloss.NewStyle().
		Width(m.w).
		Height(m.h).
		Render(m.viewport.View())
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

func (m *Model) renderTimeline() error {
	if len(m.state.messages) == 0 {
		m.viewport.SetContent("No messages yet.")
		return nil
	}

	renderedMessages := make([]string, 0, len(m.state.messages))
	for _, message := range m.state.messages {
		user := m.state.users[message.GetUserId()]
		username := traqapiext.GetUsernameOrUnknown(&user)
		timestamp := message.GetCreatedAt().Format("15:04")

		renderedContent, err := m.renderer.Render(message.GetContent())
		if err != nil {
			renderedContent = message.GetContent()
		}

		var renderedStamps string
		stamps := message.GetStamps()
		if len(stamps) > 0 {
			slices.SortStableFunc(stamps, func(a, b traqapi.MessageStamp) int {
				return a.GetCreatedAt().Compare(b.GetCreatedAt())
			})

			termProtocol := termimg.Halfblocks
			stampWidth, stampHeight, stampSpacing := 10, 4, 1
			columns := m.w / (stampWidth/2 + stampSpacing)
			gallery := termimg.NewImageGallery(columns)
			stampMap := make(map[uuid.UUID]struct{})
			for _, stamp := range stamps {
				stampID := stamp.GetStampId()
				if _, ok := stampMap[stampID]; ok {
					continue
				}

				stampMap[stampID] = struct{}{}

				img, err := m.traqContext.StampImages.Get(context.Background(), stampID)
				if err != nil {
					return fmt.Errorf("load stamp image: %w", err)
				}

				termImg := termimg.New(img)
				gallery.AddImage(termImg)
			}

			gallery.
				SetProtocol(termProtocol).
				SetImageSize(stampWidth, stampHeight).
				SetSpacing(stampSpacing)
			renderedStamps, err = gallery.Render()
			if err != nil {
				return fmt.Errorf("render stamp gallery: %w", err)
			}
		}

		renderedMessages = append(renderedMessages, lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.theme.Timeline.Time.Render(timestamp),
			m.theme.Timeline.MessageBox.
				Render(
					lipgloss.JoinVertical(
						lipgloss.Left,
						m.theme.Timeline.Username.Render("@"+username),
						renderedContent,
						"",
						renderedStamps,
					),
				),
		))

	}

	m.viewport.SetContent(
		lipgloss.JoinVertical(
			lipgloss.Left,
			renderedMessages...,
		),
	)
	m.viewport.GotoBottom()

	return nil
}
