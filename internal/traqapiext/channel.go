package traqapiext

import (
	"cmp"
	"iter"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/ras0q/bubbletree"
	"github.com/ras0q/lazytraq/internal/traqapi"
)

type ChannelNode struct {
	Channel    traqapi.Channel
	ChildNodes []*ChannelNode
	IsOpen     atomic.Bool
}

var _ bubbletree.Node[uuid.UUID] = (*ChannelNode)(nil)

// ID implements bubbletree.Tree.
func (m *ChannelNode) ID() uuid.UUID {
	return m.Channel.GetID()
}

// Content implements bubbletree.Tree.
func (m *ChannelNode) Content() string {
	prefix := ""
	if !m.IsLeaf() {
		if m.IsOpen.Load() {
			prefix = "▼"
		} else {
			prefix = "▶"
		}
	}

	return prefix + cmp.Or(m.Channel.GetName(), ".")
}

// Children implements bubbletree.Tree.
func (m *ChannelNode) Children() iter.Seq2[bubbletree.Node[uuid.UUID], bool] {
	return func(yield func(bubbletree.Node[uuid.UUID], bool) bool) {
		children := make([]*ChannelNode, 0, len(m.ChildNodes))
		excludeArchived := true // TODO: make this configurable
		if excludeArchived {
			if m.Channel.GetArchived() {
				return
			}

			for _, child := range m.ChildNodes {
				if !child.Channel.GetArchived() {
					children = append(children, child)
				}
			}
		} else {
			children = m.ChildNodes
		}

		if !m.IsOpen.Load() {
			return
		}

		for i, child := range children {
			hasNext := i < len(children)-1
			if !yield(child, hasNext) {
				return
			}
		}
	}
}

func (m *ChannelNode) IsLeaf() bool {
	return len(m.ChildNodes) == 0
}

func (m *ChannelNode) Search(id uuid.UUID) (*ChannelNode, bool) {
	if m.Channel.GetID() == id {
		return m, true
	}

	for _, child := range m.ChildNodes {
		if child.Channel.GetID() == id {
			return child, true
		}

		result, ok := child.Search(id)
		if ok {
			return result, true
		}
	}

	return nil, false

}

func ConstructTree(channels []traqapi.Channel) *ChannelNode {
	channelMap := make(map[uuid.UUID]*ChannelNode)
	var roots []*ChannelNode

	for _, channel := range channels {
		channelMap[channel.GetID()] = &ChannelNode{
			Channel:    channel,
			ChildNodes: []*ChannelNode{},
		}
	}

	for _, channel := range channels {
		channelID := channel.GetID()
		parentID, ok := channel.GetParentId().Get()
		if !ok {
			roots = append(roots, channelMap[channelID])
			continue
		}

		if parentItem, exists := channelMap[parentID]; exists {
			parentItem.ChildNodes = append(parentItem.ChildNodes, channelMap[channelID])
		}
	}

	node := ChannelNode{
		ChildNodes: roots,
		IsOpen:     atomic.Bool{},
	}
	node.IsOpen.Store(true)

	return &node
}
