package messageinput

import (
	"context"
	"errors"
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/ras0q/lazytraq/internal/traqapi"
	"github.com/ras0q/lazytraq/internal/tui/shared"
)

type Model struct {
	w, h       int
	traqClient *traqapi.Client
	textarea   textarea.Model

	channelID uuid.UUID
}

// New creates a new message input model.
func New(w, h int, traqClient *traqapi.Client) *Model {
	ta := textarea.New()
	ta.Placeholder = "Send a message with Ctrl+J..."
	ta.SetWidth(w)
	ta.SetHeight(h)
	ta.ShowLineNumbers = false
	ta.Prompt = ""

	return &Model{
		w:          w,
		h:          h,
		traqClient: traqClient,
		textarea:   ta,
	}
}

func (m *Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.textarea.Blur()
			return m, func() tea.Msg {
				return shared.ReturnToSidebarMsg{}
			}

		case "ctrl+j":
			content := m.textarea.Value()
			m.textarea.SetValue("")
			return m, m.sendMessageCmd(context.Background(), m.channelID, content)
		}

	case shared.FocusMessageInputMsg:
		m.channelID = msg.ChannelID
		m.textarea.Focus()
	}

	m.textarea, cmd = m.textarea.Update(msg)

	return m, cmd
}

func (m *Model) View() string {
	return lipgloss.NewStyle().
		Width(m.w).
		Height(m.h).
		Render(
			m.textarea.View(),
		)
}

func (m *Model) sendMessageCmd(ctx context.Context, channelID uuid.UUID, content string) tea.Cmd {
	return func() tea.Msg {
		res, err := m.traqClient.PostMessage(
			ctx,
			traqapi.NewOptPostMessageRequest(traqapi.PostMessageRequest{
				Content: content,
				Embed:   traqapi.NewOptBool(true),
			}),
			traqapi.PostMessageParams{
				ChannelId: channelID,
			},
		)
		if err != nil {
			return shared.ErrorMsg(fmt.Errorf("post message to traQ: %w", err))
		}

		switch res := res.(type) {
		case *traqapi.Message:
			return shared.MessageSentMsg{
				MessageID: res.ID,
			}

		case *traqapi.PostMessageBadRequest:
			return shared.ErrorMsg(errors.New("bad request"))

		case *traqapi.PostMessageNotFound:
			return shared.ErrorMsg(errors.New("not found"))

		default:
			return shared.ErrorMsg(fmt.Errorf("unreachable error"))
		}
	}
}
