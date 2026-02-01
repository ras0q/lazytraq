package tui

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/ras0q/lazytraq/internal/traqapi"
	"github.com/ras0q/lazytraq/internal/traqapiext"
	"github.com/ras0q/lazytraq/internal/tui/shared"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/content"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/header"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/messageinput"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/preview"
	"github.com/ras0q/lazytraq/internal/tui/viewmodel/sidebar"
)

type AppModel struct {
	header       *header.Model
	sidebar      *sidebar.Model
	content      *content.Model
	messageInput *messageinput.Model
	preview      *preview.Model
	Errors       []error

	focus     focusArea
	channelID uuid.UUID
}

type focusArea int

const (
	focusAreaHeader focusArea = iota + 1
	focusAreaSidebar
	focusAreaContent
	focusAreaMessageInput
	focusAreaPreview
)

func NewAppModel(w, h int, apiHost string, securitySource *traqapiext.SecuritySource) (*AppModel, error) {
	httpClient := http.DefaultClient
	httpClient.Timeout = 10 * time.Second
	traqClient, err := traqapi.NewClient(
		fmt.Sprintf("https://%s/api/v3", apiHost),
		securitySource,
		traqapi.WithClient(httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("create traq client: %w", err)
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
	messageInputWidth := contentWidth
	messageInputHeight := mainHeight - contentHeight
	previewWidth := w - sidebarWidth - contentWidth
	previewHeight := mainHeight
	bp := 2

	return &AppModel{
		header: header.New(
			headerWidth-bp,
			headerHeight-bp,
			apiHost,
		),
		sidebar: sidebar.New(
			sidebarWidth-bp,
			sidebarHeight-bp,
			traqClient,
		),
		content: content.New(
			contentWidth-bp,
			contentHeight-bp,
			traqClient,
		),
		messageInput: messageinput.New(
			messageInputWidth-bp,
			messageInputHeight-bp,
			traqClient,
		),
		preview: preview.New(
			previewWidth-bp,
			previewHeight-bp,
			traqClient,
		),
		Errors:    make([]error, 0, 10),
		focus:     focusAreaSidebar,
		channelID: uuid.Nil,
	}, nil
}

var _ tea.Model = (*AppModel)(nil)

func (m *AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.header.Init(),
		m.sidebar.Init(),
		m.content.Init(),
		m.messageInput.Init(),
		m.preview.Init(),
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
		channelID := msg.ID
		if channelID == uuid.Nil {
			break
		}

		m.focus = focusAreaContent
		m.channelID = channelID

		cmd := m.content.FetchMessagesCmd(context.Background(), channelID)
		cmds = append(cmds, cmd)

	case shared.MessageSentMsg:
		if m.channelID == uuid.Nil {
			break
		}

		cmd := m.content.FetchMessagesCmd(context.Background(), m.channelID)
		cmds = append(cmds, cmd)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "n":
			m.focus = focusAreaMessageInput
			cmds = append(cmds, func() tea.Msg {
				return shared.FocusMessageInputMsg{
					ChannelID: m.channelID,
				}
			})

		default:
			switch m.focus {
			case focusAreaHeader:
				_header, cmd := m.header.Update(msg)
				m.header = _header.(*header.Model)
				cmds = append(cmds, cmd)

			case focusAreaSidebar:
				_sidebar, cmd := m.sidebar.Update(msg)
				m.sidebar = _sidebar.(*sidebar.Model)
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
			}
		}

	default:
		_header, cmd := m.header.Update(msg)
		m.header = _header.(*header.Model)
		cmds = append(cmds, cmd)

		_sidebar, cmd := m.sidebar.Update(msg)
		m.sidebar = _sidebar.(*sidebar.Model)
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
	}

	return m, tea.Batch(cmds...)
}

func (m *AppModel) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		withBorder(m.header.View(), m.focus == focusAreaHeader),
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			withBorder(m.sidebar.View(), m.focus == focusAreaSidebar),
			lipgloss.JoinVertical(
				lipgloss.Left,
				withBorder(m.content.View(), m.focus == focusAreaContent),
				withBorder(m.messageInput.View(), m.focus == focusAreaMessageInput),
			),
			withBorder(m.preview.View(), m.focus == focusAreaPreview),
		),
	)
}

var borderStyle = lipgloss.NewStyle().
	Border(lipgloss.DoubleBorder())

var focusedBorderStyle = borderStyle.
	BorderForeground(lipgloss.Color("205"))

func withBorder(s string, focused bool) string {
	if focused {
		return focusedBorderStyle.Render(s)
	}

	return borderStyle.Render(s)
}
