package preview

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/sixel"
	"github.com/google/uuid"
	"github.com/motoki317/sc"
	"github.com/ras0q/lazytraq/internal/traqapi"
	"github.com/ras0q/lazytraq/internal/traqapiext"
	"github.com/ras0q/lazytraq/internal/tui/shared"
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"
	"golang.org/x/image/draw"
	"golang.org/x/sync/errgroup"
)

type (
	stampsRenderedMsg string
)

var (
	renderedStampsStyle = lipgloss.NewStyle().Height(5)
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

		resizedImg := image.NewRGBA(image.Rect(0, 0, 32, 32))
		draw.BiLinear.Scale(resizedImg, resizedImg.Bounds(), img, img.Bounds(), draw.Src, nil)

		return resizedImg, nil
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
		m.renderedStamps = renderSixelImage(msg.RenderedStamps, 2, m.w)

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

			hgap := 8
			totalWidth := -hgap
			maxHeight := 0
			chunks := make([]image.Image, 0, len(message.Message.Stamps))

			stampMap := make(map[uuid.UUID]struct{})
			for _, stamp := range message.Message.Stamps {
				stampID := stamp.GetStampId()
				if _, ok := stampMap[stampID]; ok {
					continue
				}

				stampMap[stampID] = struct{}{}

				chunk, err := m.stampImageCache.Get(context.Background(), stampID)
				if err != nil {
					return fmt.Errorf("load stamp image: %w", err)
				}

				chunks = append(chunks, chunk)

				totalWidth += chunk.Bounds().Dx() + hgap
				if h := chunk.Bounds().Dy(); h > maxHeight {
					maxHeight = h
				}
			}

			mergedImg := image.NewRGBA(image.Rect(0, 0, totalWidth, maxHeight))
			currentX := 0
			for _, chunk := range chunks {
				drawRect := image.Rect(currentX, 0, currentX+chunk.Bounds().Dx(), chunk.Bounds().Dy())
				draw.Draw(mergedImg, drawRect, chunk, chunk.Bounds().Min, draw.Src)
				currentX += chunk.Bounds().Dx() + hgap
			}

			var buf bytes.Buffer
			e := sixel.Encoder{}
			if err := e.Encode(&buf, mergedImg); err != nil {
				return fmt.Errorf("encode image chunk to sixel: %w", err)
			}

			renderedStamps = ansi.SixelGraphics(0, 1, 0, buf.Bytes())

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

func renderSixelImage(seq string, h, w int) string {
	var buf bytes.Buffer
	// Overwrite the target area with a visible glyph (full block '█') to
	// cover any sixel image previously rendered. Many terminals composite
	// sixel graphics behind the text layer, so printing plain spaces may
	// not visually hide the image. Writing a visible character ensures the
	// text layer replaces the image and prevents burn-in.
	for range h {
		for range w {
			buf.WriteRune('█')
		}
		buf.WriteByte('\n')
	}

	buf.WriteString(ansi.CursorUp(h))
	buf.WriteString(seq)
	buf.WriteString(ansi.CursorUp(h))

	return buf.String()
}
