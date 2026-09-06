package agents

import (
	"github.com/google/uuid"
)

// ParseUUID is a helper to parse UUID strings
func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
