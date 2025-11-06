package shared

import (
	"github.com/google/uuid"
)

type (
	ErrorMsg error

	ReturnToSidebarMsg struct{}

	OpenChannelMsg struct {
		ID uuid.UUID
	}
)
