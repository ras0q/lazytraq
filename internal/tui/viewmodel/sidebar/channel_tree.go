package sidebar

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/google/uuid"
	"github.com/ras0q/bubbletree"
	"github.com/ras0q/lazytraq/internal/traqapi"
	"github.com/ras0q/lazytraq/internal/traqapiext"
)

type ChannelTreeModel struct {
	w, h       int
	traqClient *traqapi.Client
	treeModel  bubbletree.Model[uuid.UUID]

	currentTree *traqapiext.ChannelNode
}

func NewChannelsModel(w, h int, traqClient *traqapi.Client) *ChannelTreeModel {
	model := &ChannelTreeModel{
		w:          w,
		h:          h,
		traqClient: traqClient,
		treeModel:  bubbletree.New[uuid.UUID](w, h),
	}
	model.treeModel.OnUpdate = model.OnTreeUpdate

	return model
}

var _ tea.Model = (*ChannelTreeModel)(nil)

func (m *ChannelTreeModel) Init() tea.Cmd {
	ctx := context.Background()
	return m.getChannelsCmd(ctx)
}

func (m *ChannelTreeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0, 2)

	switch msg := msg.(type) {
	case *traqapi.ChannelList:
		publicChannels := msg.Public
		tree := traqapiext.ConstructTree(publicChannels)
		m.currentTree = tree
		cmd := m.treeModel.SetTree(tree)
		cmds = append(cmds, cmd)
	}

	var cmd tea.Cmd

	m.treeModel, cmd = m.treeModel.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *ChannelTreeModel) View() string {
	return lipgloss.NewStyle().
		Width(m.w).
		Height(m.h).
		Render(
			m.treeModel.View(),
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

func (m *ChannelTreeModel) OnTreeUpdate(renderedLines []bubbletree.RenderedLine[uuid.UUID], focusedID uuid.UUID, msg tea.Msg) tea.Cmd {
	if m.currentTree == nil {
		return nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "h":
			channelNode, ok := m.currentTree.Search(focusedID)
			if !ok {
				break
			}

			var newFocusedID uuid.UUID
			if channelNode.IsOpen.Load() {
				channelNode.IsOpen.Store(false)
				newFocusedID = focusedID
			} else {
				parentID, ok := channelNode.Channel.GetParentId().Get()
				if ok {
					parentNode, ok := m.currentTree.Search(parentID)
					if !ok {
						break
					}

					parentNode.IsOpen.Store(false)
					newFocusedID = parentID
				} else {
					// Fold root channel
					m.currentTree.IsOpen.Store(false)
				}
			}

			cmd = tea.Batch(
				m.treeModel.SetTree(m.currentTree),
				m.treeModel.SetFocusedID(newFocusedID),
			)

		case "l":
			channelNode, ok := m.currentTree.Search(focusedID)
			if !ok {
				break
			}

			if channelNode.IsLeaf() {
				break
			}

			channelNode.IsOpen.Store(true)

			cmd = tea.Batch(
				m.treeModel.SetTree(m.currentTree),
				m.treeModel.SetFocusedID(focusedID),
			)
		}
	}

	return cmd
}
