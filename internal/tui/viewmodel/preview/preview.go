package preview

import (
	"context"
	"fmt"
	"image"
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
	"github.com/motoki317/sc"
	"github.com/ras0q/lazytraq/internal/traqapi"
	"github.com/ras0q/lazytraq/internal/traqapiext"
	"github.com/ras0q/lazytraq/internal/tui/shared"
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"
	"golang.org/x/sync/errgroup"
)

var (
	renderedStampsStyle = lipgloss.NewStyle().Height(8)
)

type Model struct {
	w, h            int
	traqClient      *traqapi.Client
	viewport        viewport.Model
	renderer        *glamour.TermRenderer
	stampImageCache *sc.Cache[uuid.UUID, image.Image]

	message        *traqapiext.MessageItem
	renderedStamps string
}

var _ tea.Model = (*Model)(nil)

func New(w, h int, traqClient *traqapi.Client) *Model {
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(w),
	)

	viewport := viewport.New(w, h-10)
	viewport.SetContent("No message selected.")

	stampImageCache := sc.NewMust(func(ctx context.Context, stampID uuid.UUID) (image.Image, error) {
		res, err := traqClient.GetStampImage(ctx, traqapi.GetStampImageParams{
			StampId: stampID,
		})
		if err != nil {
			return nil, fmt.Errorf("get stamp image from traQ: %w", err)
		}

		var img image.Image
		switch res := res.(type) {
		case *traqapi.GetStampImageNotFound:
			return nil, fmt.Errorf("stamp nof found: %w", err)

		case *traqapi.GetStampImageOKImageGIF:
			img, _, err = image.Decode(res.Data)
			if err != nil {
				return nil, fmt.Errorf("decode file to image: %w", err)
			}

		case *traqapi.GetStampImageOKImageJpeg:
			img, _, err = image.Decode(res.Data)
			if err != nil {
				return nil, fmt.Errorf("decode file to image: %w", err)
			}

		case *traqapi.GetStampImageOKImagePNG:
			img, _, err = image.Decode(res.Data)
			if err != nil {
				return nil, fmt.Errorf("decode file to image: %w", err)
			}

		case *traqapi.GetStampImageOKImageSvgXML:
			c, err := canvas.ParseSVG(res.Data)
			if err != nil {
				return nil, fmt.Errorf("parse svg: %w", err)
			}

			img = rasterizer.Draw(c, 96.0, canvas.DefaultColorSpace)
		}

		return img, nil
	}, 1*time.Hour, 2*time.Hour)

	return &Model{
		w:               w,
		h:               h - 1,
		traqClient:      traqClient,
		viewport:        viewport,
		renderer:        renderer,
		stampImageCache: stampImageCache,
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
		m.renderedStamps = msg.RenderedStamps

	case shared.PreviewMessageMsg:
		m.message = msg
		m.viewport.SetContent("")
		m.renderedStamps = ""
		cmds = append(cmds, m.renderMessageCmd(msg))

	case tea.KeyMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model.
func (m *Model) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewport.View(),
		renderedStampsStyle.Render(m.renderedStamps),
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

				img, err := m.stampImageCache.Get(context.Background(), stampID)
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
