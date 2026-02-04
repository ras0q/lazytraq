package shared

import (
	"github.com/google/uuid"
	"github.com/ras0q/lazytraq/internal/traqapi"
)

type (
	ErrorMsg error

	ReturnToSidebarMsg struct{}

	OpenChannelMsg struct {
		Target *traqapi.Channel
	}

	FocusMessageInputMsg struct {
		ChannelID uuid.UUID
	}
	MessageSentMsg struct {
		MessageID uuid.UUID
	}
)
