package audit

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAuditEntry_Fields(t *testing.T) {
	id := uuid.New()
	actorID := uuid.New()
	resourceID := uuid.New()
	now := time.Now()

	entry := AuditEntry{
		ID:           id,
		ActorType:    "provider",
		ActorID:      actorID,
		ResourceType: "appointment",
		ResourceID:   resourceID,
		Action:       "create",
		Metadata:     map[string]string{"client": "John"},
		CreatedAt:    now,
	}

	assert.Equal(t, id, entry.ID)
	assert.Equal(t, "provider", entry.ActorType)
	assert.Equal(t, actorID, entry.ActorID)
	assert.Equal(t, "appointment", entry.ResourceType)
	assert.Equal(t, resourceID, entry.ResourceID)
	assert.Equal(t, "create", entry.Action)
	assert.Equal(t, "John", entry.Metadata["client"])
	assert.Equal(t, now, entry.CreatedAt)
}

func TestAuditEntry_NilMetadata(t *testing.T) {
	entry := AuditEntry{
		ActorType:    "system",
		ActorID:      uuid.New(),
		ResourceType: "provider",
		ResourceID:   uuid.New(),
		Action:       "deactivate",
	}

	assert.Nil(t, entry.Metadata)
}

func TestListFilter_Defaults(t *testing.T) {
	filter := ListFilter{}

	assert.Empty(t, filter.ActorType)
	assert.Nil(t, filter.ActorID)
	assert.Empty(t, filter.ResourceType)
	assert.Nil(t, filter.ResourceID)
	assert.Empty(t, filter.Action)
	assert.Zero(t, filter.Page)
	assert.Zero(t, filter.PerPage)
}

func TestListFilter_WithValues(t *testing.T) {
	actorID := uuid.New()
	resourceID := uuid.New()

	filter := ListFilter{
		ActorType:    "provider",
		ActorID:      &actorID,
		ResourceType: "appointment",
		ResourceID:   &resourceID,
		Action:       "update",
		Page:         2,
		PerPage:      50,
	}

	assert.Equal(t, "provider", filter.ActorType)
	assert.Equal(t, &actorID, filter.ActorID)
	assert.Equal(t, "appointment", filter.ResourceType)
	assert.Equal(t, &resourceID, filter.ResourceID)
	assert.Equal(t, "update", filter.Action)
	assert.Equal(t, 2, filter.Page)
	assert.Equal(t, 50, filter.PerPage)
}
