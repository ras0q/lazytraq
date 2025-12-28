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

	FocusMessageInputMsg struct {
		ChannelID uuid.UUID
	}
	MessageSentMsg struct {
		MessageID uuid.UUID
	}

	PreviewMessageMsg         *traqapiext.MessageItem
	PreviewMessageRenderedMsg struct {
		MessageID       uuid.UUID
		RenderedContent string
		RenderedStamps  string
	}
)
