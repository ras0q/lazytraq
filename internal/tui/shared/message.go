package shared

import (
	"github.com/google/uuid"
)

type (
	ReturnToSidebarMsg struct{}

	OpenChannelMsg struct {
		ID uuid.UUID
	}
)
