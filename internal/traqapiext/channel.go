package traqapiext

import (
	"cmp"
	"iter"

	"github.com/google/uuid"
	"github.com/ras0q/bubbletree"
	"github.com/ras0q/lazytraq/internal/traqapi"
)

type ChannelItem struct {
	Channel    traqapi.Channel
	ChildItems []*ChannelItem
}

var _ bubbletree.Node[uuid.UUID] = ChannelItem{}

// ID implements bubbletree.Tree.
func (m ChannelItem) ID() uuid.UUID {
	return m.Channel.GetID()
}

// Content implements bubbletree.Tree.
func (m ChannelItem) Content() string {
	return cmp.Or(m.Channel.GetName(), ".")
}

// Children implements bubbletree.Tree.
func (m ChannelItem) Children() iter.Seq2[bubbletree.Node[uuid.UUID], bool] {
	return func(yield func(bubbletree.Node[uuid.UUID], bool) bool) {
		children := make([]*ChannelItem, 0, len(m.ChildItems))
		excludeArchived := true // TODO: make this configurable
		if excludeArchived {
			if m.Channel.GetArchived() {
				return
			}

			for _, child := range m.ChildItems {
				if !child.Channel.GetArchived() {
					children = append(children, child)
				}
			}
		} else {
			children = m.ChildItems
		}

		for i, child := range children {
			hasNext := i < len(children)-1
			if !yield(child, hasNext) {
				return
			}
		}
	}
}

func ConstructTree(channels []traqapi.Channel) ChannelItem {
	channelMap := make(map[uuid.UUID]*ChannelItem)
	var roots []*ChannelItem

	for _, channel := range channels {
		channelMap[channel.GetID()] = &ChannelItem{
			Channel:    channel,
			ChildItems: []*ChannelItem{},
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
			parentItem.ChildItems = append(parentItem.ChildItems, channelMap[channelID])
		}
	}

	return ChannelItem{
		ChildItems: roots,
	}
}
