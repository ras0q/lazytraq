package messageinput

import (
	"context"
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/kujtimiihoxha/vimtea"
	"github.com/ras0q/lazytraq/internal/traqapi"
	"github.com/ras0q/lazytraq/internal/traqapiext"
	"github.com/ras0q/lazytraq/internal/tui/shared"
)

type State struct {
	channelID uuid.UUID
}

type Model struct {
	w, h        int
	traqContext *traqapiext.Context
	editor      vimtea.Editor

	state State
}

// New creates a new message input model.
func New(w, h int, traqContext *traqapiext.Context) *Model {
	editor := vimtea.NewEditor(
		vimtea.WithEnableStatusBar(true),
	)

	m := &Model{
		w:           w,
		h:           h,
		traqContext: traqContext,
		editor:      editor,
	}

	editor.AddBinding(vimtea.KeyBinding{
		Key:         "enter",
		Mode:        vimtea.ModeNormal,
		Description: "Send message",
		Handler: func(b vimtea.Buffer) tea.Cmd {
			content := b.Text()
			if len(content) == 0 {
				return nil
			}

			for i, line := range b.Lines() {
				b.DeleteAt(i, 0, i, len(line))
			}
			return m.sendMessageCmd(context.Background(), m.state.channelID, content)
		},
	})

	return m
}

func (m *Model) Init() tea.Cmd {
	_, cmd := m.editor.SetSize(m.w, m.h)
	return cmd
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.editor.GetMode() == vimtea.ModeNormal {
				return m, func() tea.Msg {
					return shared.ReturnToSidebarMsg{}
				}
			}

			m.editor.SetMode(vimtea.ModeNormal)
		}

	case shared.FocusMessageInputMsg:
		m.state.channelID = msg.ChannelID
		m.editor.SetMode(vimtea.ModeInsert)
	}

	_editor, cmd := m.editor.Update(msg)
	m.editor = _editor.(vimtea.Editor)

	return m, cmd
}

func (m *Model) View() string {
	return lipgloss.NewStyle().
		Width(m.w).
		Height(m.h).
		Render(
			m.editor.View(),
		)
}

func (m *Model) sendMessageCmd(ctx context.Context, channelID uuid.UUID, content string) tea.Cmd {
	return func() tea.Msg {
		res, err := m.traqContext.PostMessage(
			ctx,
			traqapi.PostMessageRequest{
				Content: content,
				Embed:   traqapi.NewOptBool(true),
			},
			channelID,
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
			return shared.ErrorMsg(errors.New("post message to traQ: bad request"))

		case *traqapi.PostMessageNotFound:
			return shared.ErrorMsg(errors.New("post message to traQ: not found"))

		default:
			return shared.ErrorMsg(errors.New("post message to traQ: unreachable error"))
		}
	}
}
