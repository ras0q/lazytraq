package traqapiext

import (
	"iter"

	"github.com/google/uuid"
	"github.com/ras0q/bubbletree"
	"github.com/ras0q/lazytraq/internal/traqapi"
)

type ChannelItem struct {
	Channel    traqapi.Channel
	ChildItems []ChannelItem
}

var _ bubbletree.Node[uuid.UUID] = ChannelItem{}

// ID implements bubbletree.Tree.
func (m ChannelItem) ID() uuid.UUID {
	return m.Channel.GetID()
}

// Content implements bubbletree.Tree.
func (m ChannelItem) Content() string {
	return m.Channel.GetName()
}

// Children implements bubbletree.Tree.
func (m ChannelItem) Children() iter.Seq2[bubbletree.Node[uuid.UUID], bool] {
	return func(yield func(bubbletree.Node[uuid.UUID], bool) bool) {
		for i, child := range m.ChildItems {
			hasNext := i < len(m.ChildItems)-1
			if !yield(child, hasNext) {
				return
			}
		}
	}
}

func ConstructTree(channels []traqapi.Channel) ChannelItem {
	channelMap := make(map[uuid.UUID]*ChannelItem)
	var roots []ChannelItem

	for _, channel := range channels {
		channelCopy := channel // Create a copy to avoid referencing the loop variable
		channelMap[channel.GetID()] = &ChannelItem{
			Channel:    channelCopy,
			ChildItems: []ChannelItem{},
		}
	}

	for _, channel := range channels {
		channelID := channel.GetID()
		parentID, ok := channel.GetParentId().Get()
		if !ok {
			roots = append(roots, *channelMap[channelID])
			continue
		}

		if parentItem, exists := channelMap[parentID]; exists {
			parentItem.ChildItems = append(parentItem.ChildItems, *channelMap[channelID])
		}
	}

	return ChannelItem{
		ChildItems: roots,
	}
}
