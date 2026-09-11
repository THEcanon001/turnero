package audit_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/THEcanon001/turnero/internal/audit"
	"github.com/THEcanon001/turnero/internal/testutil"
)

func TestRepository_Log(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := audit.NewRepository(pool)
	ctx := context.Background()

	t.Run("inserts entry with metadata and populates ID and CreatedAt", func(t *testing.T) {
		entry := &audit.AuditEntry{
			ActorType:    "provider",
			ActorID:      uuid.New(),
			ResourceType: "appointment",
			ResourceID:   uuid.New(),
			Action:       "create",
			Metadata:     map[string]string{"key": "value", "source": "test"},
		}

		err := repo.Log(ctx, entry)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, entry.ID)
		assert.False(t, entry.CreatedAt.IsZero())
	})

	t.Run("inserts entry with nil metadata without error", func(t *testing.T) {
		entry := &audit.AuditEntry{
			ActorType:    "employee",
			ActorID:      uuid.New(),
			ResourceType: "schedule",
			ResourceID:   uuid.New(),
			Action:       "update",
			Metadata:     nil,
		}

		err := repo.Log(ctx, entry)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, entry.ID)
		assert.False(t, entry.CreatedAt.IsZero())
	})

	t.Run("inserts entry with empty metadata map without error", func(t *testing.T) {
		entry := &audit.AuditEntry{
			ActorType:    "system",
			ActorID:      uuid.New(),
			ResourceType: "provider",
			ResourceID:   uuid.New(),
			Action:       "deactivate",
			Metadata:     map[string]string{},
		}

		err := repo.Log(ctx, entry)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, entry.ID)
	})

	t.Run("each log call returns a unique ID", func(t *testing.T) {
		actorID := uuid.New()
		resourceID := uuid.New()

		e1 := &audit.AuditEntry{
			ActorType: "provider", ActorID: actorID,
			ResourceType: "service", ResourceID: resourceID,
			Action: "delete",
		}
		e2 := &audit.AuditEntry{
			ActorType: "provider", ActorID: actorID,
			ResourceType: "service", ResourceID: resourceID,
			Action: "delete",
		}

		require.NoError(t, repo.Log(ctx, e1))
		require.NoError(t, repo.Log(ctx, e2))

		assert.NotEqual(t, e1.ID, e2.ID)
	})
}

// seedEntries inserts n entries with the given overrides applied to a base template.
// It returns the inserted entries.
func seedEntries(t *testing.T, repo *audit.Repository, ctx context.Context, entries []audit.AuditEntry) []audit.AuditEntry {
	t.Helper()
	for i := range entries {
		require.NoError(t, repo.Log(ctx, &entries[i]))
	}
	return entries
}

func TestRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := audit.NewRepository(pool)
	ctx := context.Background()

	// Seed a consistent set of entries for filter tests.
	providerActor := uuid.New()
	employeeActor := uuid.New()
	appointmentResource := uuid.New()
	serviceResource := uuid.New()

	entries := []audit.AuditEntry{
		{ActorType: "provider", ActorID: providerActor, ResourceType: "appointment", ResourceID: appointmentResource, Action: "create", Metadata: map[string]string{"env": "test"}},
		{ActorType: "provider", ActorID: providerActor, ResourceType: "appointment", ResourceID: appointmentResource, Action: "update"},
		{ActorType: "employee", ActorID: employeeActor, ResourceType: "appointment", ResourceID: appointmentResource, Action: "complete"},
		{ActorType: "employee", ActorID: employeeActor, ResourceType: "service", ResourceID: serviceResource, Action: "create"},
		{ActorType: "system", ActorID: uuid.New(), ResourceType: "service", ResourceID: serviceResource, Action: "update"},
		{ActorType: "system", ActorID: uuid.New(), ResourceType: "provider", ResourceID: uuid.New(), Action: "deactivate"},
	}
	seedEntries(t, repo, ctx, entries)

	t.Run("no filters returns all entries with default pagination", func(t *testing.T) {
		result, total, err := repo.List(ctx, audit.ListFilter{})
		require.NoError(t, err)
		assert.Equal(t, 6, total)
		assert.Len(t, result, 6)
	})

	t.Run("filter by ActorType provider", func(t *testing.T) {
		result, total, err := repo.List(ctx, audit.ListFilter{ActorType: "provider"})
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, result, 2)
		for _, e := range result {
			assert.Equal(t, "provider", e.ActorType)
		}
	})

	t.Run("filter by ActorID", func(t *testing.T) {
		result, total, err := repo.List(ctx, audit.ListFilter{ActorID: &employeeActor})
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, result, 2)
		for _, e := range result {
			assert.Equal(t, employeeActor, e.ActorID)
		}
	})

	t.Run("filter by ResourceType appointment", func(t *testing.T) {
		result, total, err := repo.List(ctx, audit.ListFilter{ResourceType: "appointment"})
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, result, 3)
		for _, e := range result {
			assert.Equal(t, "appointment", e.ResourceType)
		}
	})

	t.Run("filter by ResourceID", func(t *testing.T) {
		result, total, err := repo.List(ctx, audit.ListFilter{ResourceID: &serviceResource})
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, result, 2)
		for _, e := range result {
			assert.Equal(t, serviceResource, e.ResourceID)
		}
	})

	t.Run("filter by Action create", func(t *testing.T) {
		result, total, err := repo.List(ctx, audit.ListFilter{Action: "create"})
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, result, 2)
		for _, e := range result {
			assert.Equal(t, "create", e.Action)
		}
	})

	t.Run("filter by multiple fields combined", func(t *testing.T) {
		result, total, err := repo.List(ctx, audit.ListFilter{
			ActorType:    "provider",
			ActorID:      &providerActor,
			ResourceType: "appointment",
			ResourceID:   &appointmentResource,
			Action:       "create",
		})
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		require.Len(t, result, 1)
		assert.Equal(t, "provider", result[0].ActorType)
		assert.Equal(t, "create", result[0].Action)
		// Metadata must round-trip correctly.
		assert.Equal(t, map[string]string{"env": "test"}, result[0].Metadata)
	})

	t.Run("filter returns zero results for unknown actor type", func(t *testing.T) {
		result, total, err := repo.List(ctx, audit.ListFilter{ActorType: "ghost"})
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Empty(t, result)
	})

	t.Run("pagination page 1 perPage 2 returns 2 results and correct total", func(t *testing.T) {
		result, total, err := repo.List(ctx, audit.ListFilter{Page: 1, PerPage: 2})
		require.NoError(t, err)
		assert.Equal(t, 6, total)
		assert.Len(t, result, 2)
	})

	t.Run("pagination page 2 perPage 2 returns next 2 results", func(t *testing.T) {
		result, total, err := repo.List(ctx, audit.ListFilter{Page: 2, PerPage: 2})
		require.NoError(t, err)
		assert.Equal(t, 6, total)
		assert.Len(t, result, 2)
	})

	t.Run("pagination page 3 perPage 2 returns remaining results", func(t *testing.T) {
		result, total, err := repo.List(ctx, audit.ListFilter{Page: 3, PerPage: 2})
		require.NoError(t, err)
		assert.Equal(t, 6, total)
		assert.Len(t, result, 2)
	})

	t.Run("page 0 and perPage 0 default to page 1 and perPage 20", func(t *testing.T) {
		// Page=0 → 1, PerPage=0 → 20 per repository logic.
		result, total, err := repo.List(ctx, audit.ListFilter{Page: 0, PerPage: 0})
		require.NoError(t, err)
		assert.Equal(t, 6, total)
		// All 6 entries fit within the default perPage of 20.
		assert.Len(t, result, 6)
	})

	t.Run("results are ordered by created_at descending", func(t *testing.T) {
		result, _, err := repo.List(ctx, audit.ListFilter{})
		require.NoError(t, err)
		require.True(t, len(result) >= 2, "need at least 2 results to check ordering")

		for i := 1; i < len(result); i++ {
			assert.True(t,
				!result[i].CreatedAt.After(result[i-1].CreatedAt),
				"entry %d has created_at after entry %d", i, i-1,
			)
		}
	})

	t.Run("metadata is preserved correctly on round-trip", func(t *testing.T) {
		// Re-query the single entry that had metadata.
		result, total, err := repo.List(ctx, audit.ListFilter{
			ActorType: "provider",
			Action:    "create",
		})
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		require.Len(t, result, 1)
		assert.Equal(t, map[string]string{"env": "test"}, result[0].Metadata)
	})

	t.Run("entry with nil metadata has nil Metadata field after list", func(t *testing.T) {
		// The second provider entry has nil metadata.
		result, _, err := repo.List(ctx, audit.ListFilter{
			ActorType: "provider",
			Action:    "update",
		})
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Nil(t, result[0].Metadata)
	})
}

func TestRepository_List_LargePage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := audit.NewRepository(pool)
	ctx := context.Background()

	// Seed exactly 5 entries.
	for i := 0; i < 5; i++ {
		e := &audit.AuditEntry{
			ActorType:    "provider",
			ActorID:      uuid.New(),
			ResourceType: "appointment",
			ResourceID:   uuid.New(),
			Action:       "create",
		}
		require.NoError(t, repo.Log(ctx, e))
		// Small sleep to guarantee distinct created_at values on fast machines.
		time.Sleep(time.Millisecond)
	}

	t.Run("page beyond last returns empty slice but correct total", func(t *testing.T) {
		result, total, err := repo.List(ctx, audit.ListFilter{Page: 99, PerPage: 10})
		require.NoError(t, err)
		assert.Equal(t, 5, total)
		assert.Empty(t, result)
	})
}
