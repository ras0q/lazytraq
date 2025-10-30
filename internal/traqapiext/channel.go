package traqapiext

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/ras0q/lazytraq/internal/traqapi"
)

type ChannelItem struct {
	Channel traqapi.Channel
}

var _ list.Item = ChannelItem{}
var _ list.DefaultItem = ChannelItem{}

// FilterValue implements list.Item.
func (m ChannelItem) FilterValue() string {
	return fmt.Sprintf(
		"%s - %s",
		m.Channel.GetName(),
		m.Channel.GetTopic(),
	)
}

// Description implements list.DefaultItem.
func (m ChannelItem) Description() string {
	return m.Channel.GetTopic()
}

// Title implements list.DefaultItem.
func (m ChannelItem) Title() string {
	return m.Channel.GetName()
}
