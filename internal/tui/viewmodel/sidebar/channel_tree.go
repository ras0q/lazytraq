package sidebar

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/ras0q/lazytraq/internal/traqapi"
	"github.com/ras0q/lazytraq/internal/traqapiext"
)

type ChannelTreeModel struct {
	w, h              int
	traqClient        *traqapi.Client
	channelsListModel list.Model
}

func NewChannelsModel(w, h int, traqClient *traqapi.Client) *ChannelTreeModel {
	return &ChannelTreeModel{
		w:          w,
		h:          h,
		traqClient: traqClient,
		channelsListModel: list.New(
			[]list.Item{},
			list.NewDefaultDelegate(),
			w,
			h,
		),
	}
}

var _ tea.Model = (*ChannelTreeModel)(nil)

func (m *ChannelTreeModel) Init() tea.Cmd {
	ctx := context.Background()
	return m.getChannelsCmd(ctx)
}

func (m *ChannelTreeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *traqapi.ChannelList:
		channels := msg.Public
		itemsLen := len(m.channelsListModel.Items())
		for i, channel := range channels {
			m.channelsListModel.InsertItem(itemsLen+i, traqapiext.ChannelItem{
				Channel: channel,
			})
			break // TODO: fix performance issue
		}

		return m, nil
	}

	cmds := make([]tea.Cmd, 0, 2)
	var cmd tea.Cmd

	m.channelsListModel, cmd = m.channelsListModel.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *ChannelTreeModel) View() string {
	return lipgloss.NewStyle().
		Width(m.w).
		Height(m.h).
		Render(
			m.channelsListModel.View(),
		)
}

func (m *ChannelTreeModel) getChannelsCmd(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		channels, err := m.traqClient.GetChannels(ctx, traqapi.GetChannelsParams{})
		if err != nil {
			return fmt.Errorf("get channels from traQ: %w", err)
		}

		return channels
	}
}
