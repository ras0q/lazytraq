package shared

import (
	"github.com/google/uuid"
	"github.com/ras0q/lazytraq/internal/traqapiext"
)

type (
	ErrorMsg error

	ReturnToSidebarMsg struct{}

	OpenChannelMsg struct {
		ID uuid.UUID
	}

	PreviewMessageMsg         *traqapiext.MessageItem
	PreviewMessageRenderedMsg string
)
