package tui

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ras0q/lazytraq/internal/traqapi"
	"github.com/ras0q/lazytraq/internal/traqapiext"
	"github.com/ras0q/lazytraq/internal/tui/shared"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/channelcontent"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/channeltree"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/header"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/messageinput"
)

type AppModel struct {
	theme          shared.Theme
	header         *header.Model
	channelTree    *channeltree.Model
	messageInput   *messageinput.Model
	channelContent *channelcontent.Model
	Errors         []error

	focus   focusArea
	channel *traqapi.Channel
}

type focusArea int

const (
	focusAreaHeader focusArea = iota + 1
	focusAreaSidebar
	focusAreaMessageInput
	focusAreaChannelContent
)

func NewAppModel(w, h int, apiHost string, securitySource *traqapiext.SecuritySource) (*AppModel, error) {
	traqContext, err := traqapiext.NewContext(apiHost, securitySource)
	if err != nil {
		return nil, fmt.Errorf("create traq context: %w", err)
	}

	if os.Getenv("DEBUG") != "" {
		h -= 2
	}

	// Layout calculation
	// ---------------------
	// |       header      |
	// |-------------------|
	// |    |   content    |
	// | ct |              |
	// |    |--------------|
	// |    | messageInput |
	// ---------------------

	headerHeight := 3
	mainHeight := h - headerHeight
	sidebarHeight := mainHeight
	channelContentHeight := mainHeight * 7 / 10
	messageInputHeight := mainHeight - channelContentHeight

	headerWidth := w
	sidebarWidth := w * 2 / 10
	messageInputWidth := w - sidebarWidth
	channelContentWidth := w - sidebarWidth
	padding := 2

	theme := shared.DefaultTheme()

	return &AppModel{
		theme: theme,
		header: header.New(
			headerWidth-padding,
			headerHeight-padding,
			apiHost,
			traqContext,
			theme,
		),
		channelTree: channeltree.New(
			sidebarWidth-padding,
			sidebarHeight-padding,
			traqContext,
		),
		messageInput: messageinput.New(
			messageInputWidth-padding,
			messageInputHeight-padding,
			traqContext,
		),
		channelContent: channelcontent.New(
			channelContentWidth-padding,
			channelContentHeight-padding,
			traqContext,
			theme,
		),
		Errors:  make([]error, 0, 10),
		focus:   focusAreaSidebar,
		channel: nil,
	}, nil
}

var _ tea.Model = (*AppModel)(nil)

func (m *AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.header.Init(),
		m.channelTree.Init(),
		m.messageInput.Init(),
		m.channelContent.Init(),
	)
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0, 10)

	if os.Getenv("DEBUG") != "" {
		t := fmt.Sprintf("%T", msg)
		if msg, ok := msg.(fmt.Stringer); ok {
			t = fmt.Sprintf("%s (%s)", t, msg.String())
		}
		if t != "tea.printLineMessage" {
			cmds = append(cmds, tea.Printf("%s", t))
		}
	}

	switch msg := msg.(type) {
	case shared.ErrorMsg:
		m.Errors = append(m.Errors, msg)
		return m, tea.Quit

	case shared.ReturnToSidebarMsg:
		m.focus = focusAreaSidebar

	case shared.OpenChannelMsg:
		channel := msg.Target
		if channel == nil {
			break
		}

		m.focus = focusAreaChannelContent
		m.channel = channel

		cmd := m.channelContent.FetchMessagesCmd(context.Background(), channel.ID)
		cmds = append(cmds, cmd)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "n":
			if m.channel.Force {
				break
			}

			m.focus = focusAreaMessageInput
			cmds = append(cmds, func() tea.Msg {
				return shared.FocusMessageInputMsg{
					ChannelID: m.channel.ID,
				}
			})

		default:
			switch m.focus {
			case focusAreaHeader:
				_header, cmd := m.header.Update(msg)
				m.header = _header.(*header.Model)
				cmds = append(cmds, cmd)

			case focusAreaSidebar:
				_sidebar, cmd := m.channelTree.Update(msg)
				m.channelTree = _sidebar.(*channeltree.Model)
				cmds = append(cmds, cmd)

			case focusAreaMessageInput:
				_messageInput, cmd := m.messageInput.Update(msg)
				m.messageInput = _messageInput.(*messageinput.Model)
				cmds = append(cmds, cmd)

			case focusAreaChannelContent:
				_channelContent, cmd := m.channelContent.Update(msg)
				m.channelContent = _channelContent.(*channelcontent.Model)
				cmds = append(cmds, cmd)
			}
		}

	default:
		_header, cmd := m.header.Update(msg)
		m.header = _header.(*header.Model)
		cmds = append(cmds, cmd)

		_sidebar, cmd := m.channelTree.Update(msg)
		m.channelTree = _sidebar.(*channeltree.Model)
		cmds = append(cmds, cmd)

		_messageInput, cmd := m.messageInput.Update(msg)
		m.messageInput = _messageInput.(*messageinput.Model)
		cmds = append(cmds, cmd)

		_channelContent, cmd := m.channelContent.Update(msg)
		m.channelContent = _channelContent.(*channelcontent.Model)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *AppModel) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.theme.WithBorder(m.header.View(), m.focus == focusAreaHeader),
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.theme.WithBorder(m.channelTree.View(), m.focus == focusAreaSidebar),
			lipgloss.JoinVertical(
				lipgloss.Left,
				m.theme.WithBorder(m.channelContent.View(), m.focus == focusAreaChannelContent),
				m.theme.WithBorder(m.messageInput.View(), m.focus == focusAreaMessageInput),
			),
		),
	)
}
