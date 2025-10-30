package traqapiext

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/ras0q/lazytraq/internal/traqapi"
)

type MessageItem struct {
	Message traqapi.Message
}

var _ list.Item = MessageItem{}
var _ list.DefaultItem = MessageItem{}

// FilterValue implements list.Item.
func (m MessageItem) FilterValue() string {
	return fmt.Sprintf(
		"@%s - %s",
		m.Message.GetUserId(), // TODO: display username
		m.Message.GetContent(),
	)
}

// Description implements list.DefaultItem.
func (m MessageItem) Description() string {
	return m.Message.GetContent()
}

// Title implements list.DefaultItem.
func (m MessageItem) Title() string {
	return m.Message.GetUserId().String() // TODO: display username
}
