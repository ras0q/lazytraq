package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"slices"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/google/uuid"
	"github.com/ras0q/lazytraq/internal/model/sidebar"
	"github.com/ras0q/lazytraq/internal/traqapi"
	"github.com/ras0q/lazytraq/internal/traqapiext"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
	"golang.org/x/term"
)

func main() {
	if err := runProgram(); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func runProgram() error {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return fmt.Errorf("get terminal size: %w", err)
	}

	h = h - 2

	model, err := newRootModel(w, h)
	if err != nil {
		return fmt.Errorf("create root model: %w", err)
	}

	p := tea.NewProgram(model)

	eg := errgroup.Group{}
	eg.Go(func() error {
		return <-model.errCh
	})

	eg.Go(func() error {
		if _, err := p.Run(); err != nil {
			return err
		}

		model.errCh <- nil

		return nil
	})

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("run program: %w", err)
	}

	return nil
}

type rootModel struct {
	w, h              int
	sidebar           tea.Model
	spinnerModel      spinner.Model
	messagesListModel list.Model
	traqClient        *traqapi.Client
	oauth2Token       *oauth2.Token
	errCh             chan error
}

func newRootModel(w, h int) (*rootModel, error) {
	messagesListModel := list.New(
		[]list.Item{},
		list.NewDefaultDelegate(),
		w,
		h,
	)

	traqClient, err := traqapi.NewClient(
		"https://q.trap.jp/api/v3",
		traqapiext.NewSecuritySource(),
	)
	if err != nil {
		return nil, fmt.Errorf("create traq client: %w", err)
	}

	sidebarWidth := int(float64(w) * 0.2)

	return &rootModel{
		w:                 w,
		h:                 h,
		sidebar:           sidebar.New(sidebarWidth, h),
		spinnerModel:      spinner.New(spinner.WithSpinner(spinner.Dot)),
		messagesListModel: messagesListModel,
		traqClient:        traqClient,
		oauth2Token:       nil,
		errCh:             make(chan error),
	}, nil
}

var _ tea.Model = (*rootModel)(nil)

func (m *rootModel) Init() tea.Cmd {
	return tea.Batch(
		m.sidebar.Init(),
		m.spinnerModel.Tick,
	)
}

func (m *rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	ctx := context.Background()

	switch msg := msg.(type) {
	case error:
		m.errCh <- msg
		return m, tea.Quit

	case *traqapi.GetMessagesOKHeaders:
		messages := msg.Response
		slices.Reverse(messages)

		itemsLen := len(m.messagesListModel.Items())
		for i, message := range messages {
			m.messagesListModel.InsertItem(itemsLen+i, traqapiext.MessageItem{
				Message: message,
			})
			m.messagesListModel.CursorDown()
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "r":
			return m, m.getMessagesCmd(ctx, uuid.MustParse("f58c72a4-14f0-423c-9259-dbb4a90ca35f"))
		}
	}

	cmds := make([]tea.Cmd, 0, 10)
	var cmd tea.Cmd

	m.sidebar, cmd = m.sidebar.Update(msg)
	cmds = append(cmds, cmd)

	m.spinnerModel, cmd = m.spinnerModel.Update(msg)
	cmds = append(cmds, cmd)

	m.messagesListModel, cmd = m.messagesListModel.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

var rootStyle = lipgloss.NewStyle().
	Padding(1, 2)

func (m *rootModel) View() string {
	sidebarWidth := int(float64(m.w) * 0.2)
	mainWidth := m.w - sidebarWidth

	sidebar := m.sidebar.View()

	main := rootStyle.
		Width(mainWidth).
		Height(m.h).
		Render(m.messagesListModel.View())

	return lipgloss.JoinHorizontal(
		lipgloss.Bottom,
		sidebar,
		main,
	)
}

func (m *rootModel) getMessagesCmd(ctx context.Context, channelID uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		res, err := m.traqClient.GetMessages(ctx, traqapi.GetMessagesParams{
			ChannelId: channelID,
		})
		if err != nil {
			return fmt.Errorf("get messages from traQ: %w", err)
		}

		switch res := res.(type) {
		case *traqapi.GetMessagesOKHeaders:
			return res

		case *traqapi.GetMessagesBadRequest:
			return errors.New("bad request")

		case *traqapi.GetMessagesNotFound:
			return errors.New("not found")

		default:
			return fmt.Errorf("unreachable error")
		}
	}

}
