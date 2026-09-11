package audit

import (
	"time"

	"github.com/google/uuid"
)

// AuditEntry represents a single audit log record.
type AuditEntry struct {
	ID           uuid.UUID         `json:"id"`
	ActorType    string            `json:"actor_type"`
	ActorID      uuid.UUID         `json:"actor_id"`
	ResourceType string            `json:"resource_type"`
	ResourceID   uuid.UUID         `json:"resource_id"`
	Action       string            `json:"action"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
}

// ListFilter holds filters for querying audit log entries.
type ListFilter struct {
	ActorType    string
	ActorID      *uuid.UUID
	ResourceType string
	ResourceID   *uuid.UUID
	Action       string
	Page         int
	PerPage      int
}
