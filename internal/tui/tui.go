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
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/channeltree"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/content"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/header"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/messageinput"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/preview"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/timeline"
)

type AppModel struct {
	theme        shared.Theme
	header       *header.Model
	channelTree  *channeltree.Model
	content      *content.Model
	messageInput *messageinput.Model
	preview      *preview.Model
	timeline     *timeline.Model
	Errors       []error

	focus   focusArea
	channel *traqapi.Channel
}

type focusArea int

const (
	focusAreaHeader focusArea = iota + 1
	focusAreaSidebar
	focusAreaContent
	focusAreaMessageInput
	focusAreaPreview
	focusAreaTimeline
)

func NewAppModel(w, h int, apiHost string, securitySource *traqapiext.SecuritySource) (*AppModel, error) {
	traqContext, err := traqapiext.NewContext(apiHost, securitySource)
	if err != nil {
		return nil, fmt.Errorf("create traq context: %w", err)
	}

	if os.Getenv("DEBUG") != "" {
		h -= 2
	}

	headerHeight := 3
	headerWidth := w
	mainHeight := h - headerHeight
	sidebarWidth := int(float64(w) * 0.2)
	sidebarHeight := mainHeight
	contentWidth := int(float64(w) * 0.4)
	contentHeight := int(float64(mainHeight) * 0.7)
	messageInputWidth := w - sidebarWidth
	messageInputHeight := mainHeight - contentHeight
	previewWidth := w - sidebarWidth - contentWidth
	previewHeight := mainHeight
	timelineWidth := w - sidebarWidth
	timelineHeight := contentHeight
	bp := 2

	theme := shared.DefaultTheme()

	return &AppModel{
		theme: theme,
		header: header.New(
			headerWidth-bp,
			headerHeight-bp,
			apiHost,
			traqContext,
			theme,
		),
		channelTree: channeltree.New(
			sidebarWidth-bp,
			sidebarHeight-bp,
			traqContext,
		),
		content: content.New(
			contentWidth-bp,
			contentHeight-bp,
			traqContext,
		),
		messageInput: messageinput.New(
			messageInputWidth-bp,
			messageInputHeight-bp,
			traqContext,
		),
		preview: preview.New(
			previewWidth-bp,
			previewHeight-bp,
			traqContext,
			theme,
		),
		timeline: timeline.New(
			timelineWidth-bp,
			timelineHeight-bp,
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
		m.content.Init(),
		m.messageInput.Init(),
		m.preview.Init(),
		m.timeline.Init(),
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

		m.focus = focusAreaTimeline
		m.channel = channel

		cmd := m.timeline.FetchMessagesCmd(context.Background(), channel.ID)
		cmds = append(cmds, cmd)

	case shared.MessageSentMsg:
		if m.channel == nil {
			break
		}

		cmd := m.content.FetchMessagesCmd(context.Background(), m.channel.ID)
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

			case focusAreaContent:
				_content, cmd := m.content.Update(msg)
				m.content = _content.(*content.Model)
				cmds = append(cmds, cmd)

			case focusAreaMessageInput:
				_messageInput, cmd := m.messageInput.Update(msg)
				m.messageInput = _messageInput.(*messageinput.Model)
				cmds = append(cmds, cmd)

			case focusAreaPreview:
				_preview, cmd := m.preview.Update(msg)
				m.preview = _preview.(*preview.Model)
				cmds = append(cmds, cmd)

			case focusAreaTimeline:
				_timeline, cmd := m.timeline.Update(msg)
				m.timeline = _timeline.(*timeline.Model)
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

		_content, cmd := m.content.Update(msg)
		m.content = _content.(*content.Model)
		cmds = append(cmds, cmd)

		_messageInput, cmd := m.messageInput.Update(msg)
		m.messageInput = _messageInput.(*messageinput.Model)
		cmds = append(cmds, cmd)

		_preview, cmd := m.preview.Update(msg)
		m.preview = _preview.(*preview.Model)
		cmds = append(cmds, cmd)

		_timeline, cmd := m.timeline.Update(msg)
		m.timeline = _timeline.(*timeline.Model)
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
				m.theme.WithBorder(m.timeline.View(), m.focus == focusAreaTimeline),
				m.theme.WithBorder(m.messageInput.View(), m.focus == focusAreaMessageInput),
			),
		),
	)
}
