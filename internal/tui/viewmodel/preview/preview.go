package preview

import (
	"context"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"time"

	"github.com/blacktop/go-termimg"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/ras0q/lazytraq/internal/traqapiext"
	"github.com/ras0q/lazytraq/internal/tui/shared"
	"golang.org/x/sync/errgroup"
)

type State struct {
	message        *traqapiext.MessageItem
	renderedStamps string
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
		glamour.WithWordWrap(w),
		glamour.WithStandardStyle("light"),
	)

	viewport := viewport.New(w, h-10)
	viewport.SetContent("No message selected.")

	return &Model{
		w:           w,
		h:           h - 1,
		traqContext: traqContext,
		viewport:    viewport,
		renderer:    renderer,
		theme:       theme,
	}
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	switch msg := msg.(type) {
	case shared.PreviewMessageRenderedMsg:
		time.Sleep(100 * time.Millisecond) // for smooth rendering
		m.viewport.SetContent(msg.RenderedContent)
		m.state.renderedStamps = msg.RenderedStamps

	case shared.PreviewMessageMsg:
		m.state.message = msg
		m.viewport.SetContent("")
		m.state.renderedStamps = ""
		cmds = append(cmds, m.renderMessageCmd(msg))

	case tea.KeyMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewport.View(),
		m.theme.Preview.Stamps.Render(m.state.renderedStamps),
	)
}

func (m *Model) renderMessageCmd(message *traqapiext.MessageItem) tea.Cmd {
	return func() tea.Msg {
		var (
			renderedContent string
			renderedStamps  string
		)

		eg := errgroup.Group{}

		eg.Go(func() error {
			s, err := m.renderer.Render(message.Description())
			if err != nil {
				return fmt.Errorf("render markdown: %w", err)
			}

			renderedContent = s

			return nil
		})

		eg.Go(func() error {
			if len(message.Message.Stamps) == 0 {
				return nil
			}

			termProtocol := termimg.Halfblocks
			stampWidth, stampHeight, stampSpacing := 10, 4, 1
			columns := m.w / (stampWidth/2 + stampSpacing)
			gallery := termimg.NewImageGallery(columns)
			stampMap := make(map[uuid.UUID]struct{})
			for _, stamp := range message.Message.Stamps {
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
			renderedGallery, err := gallery.Render()
			if err != nil {
				return fmt.Errorf("render stamp gallery: %w", err)
			}

			renderedStamps = renderedGallery

			return nil
		})

		if err := eg.Wait(); err != nil {
			return shared.ErrorMsg(err)
		}

		return shared.PreviewMessageRenderedMsg{
			MessageID:       message.Message.GetID(),
			RenderedContent: renderedContent,
			RenderedStamps:  renderedStamps,
		}
	}
}
