package traqapiext

import (
	"cmp"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/ras0q/lazytraq/internal/traqapi"
)

type MessageItem struct {
	Message traqapi.Message
	User    traqapi.User
}

var _ list.Item = MessageItem{}
var _ list.DefaultItem = MessageItem{}

// FilterValue implements list.Item.
func (m MessageItem) FilterValue() string {
	return fmt.Sprintf(
		"@%s - %s",
		cmp.Or(m.User.GetName(), "unknown"),
		m.Message.GetContent(),
	)
}

// Description implements list.DefaultItem.
func (m MessageItem) Description() string {
	return m.Message.GetContent()
}

// Title implements list.DefaultItem.
func (m MessageItem) Title() string {
	return fmt.Sprintf(
		"%s @%s",
		cmp.Or(m.User.GetDisplayName(), "Unknown"),
		cmp.Or(m.User.GetName(), "unknown"),
	)
}
