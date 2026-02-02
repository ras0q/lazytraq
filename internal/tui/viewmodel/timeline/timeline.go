package timeline

import (
	"context"
	"fmt"
	"slices"
	"strings"

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
		m.renderTimeline()

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

func (m *Model) renderTimeline() {
	if len(m.state.messages) == 0 {
		m.viewport.SetContent("No messages yet.")
		return
	}

	var sb strings.Builder
	separator := m.theme.Timeline.Separator.Render(" │ ")
	indent := strings.Repeat(" ", 5)

	for _, message := range m.state.messages {
		user := m.state.users[message.GetUserId()]
		username := traqapiext.GetUsernameOrUnknown(&user)
		timestamp := message.GetCreatedAt().Format("15:04")

		renderedContent, err := m.renderer.Render(message.GetContent())
		if err != nil {
			renderedContent = message.GetContent()
		}

		lines := strings.Split(strings.TrimSpace(renderedContent), "\n")
		lines = append(lines, "")

		for j, line := range lines {
			if j == 0 {
				sb.WriteString(m.theme.Timeline.Time.Render(timestamp))
				sb.WriteString(separator)
				sb.WriteString(m.theme.Timeline.Username.Render("@" + username))
				sb.WriteString("\n")
				sb.WriteString(indent)
				sb.WriteString(separator)
				sb.WriteString(line)
			} else {
				sb.WriteString(indent)
				sb.WriteString(separator)
				sb.WriteString(line)
			}
			sb.WriteString("\n")
		}
	}

	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
}
