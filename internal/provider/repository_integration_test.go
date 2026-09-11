package provider_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/THEcanon001/turnero/internal/provider"
	"github.com/THEcanon001/turnero/internal/testutil"
)

// newTestProvider returns a fully populated Provider suitable for insertion.
func newTestProvider(slug string) *provider.Provider {
	return &provider.Provider{
		Type:         provider.TypeIndividual,
		Name:         "Test Provider",
		Slug:         slug,
		Phone:        "+5491100000099",
		Timezone:     "America/Argentina/Buenos_Aires",
		Email:        slug + "@example.com",
		PasswordHash: "$2a$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ012",
	}
}

// TestRepository_Create verifies that a provider is inserted with the correct defaults.
func TestRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	repo := provider.NewRepository(pool)
	ctx := context.Background()

	p := newTestProvider("create-provider")
	err := repo.Create(ctx, p)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, p.ID, "ID should be set after create")
	assert.Equal(t, provider.PlanFree, p.Plan, "Plan should default to free")
	assert.True(t, p.IsActive, "IsActive should default to true")
	assert.False(t, p.CreatedAt.IsZero(), "CreatedAt should be set")
	assert.False(t, p.UpdatedAt.IsZero(), "UpdatedAt should be set")
}

// TestRepository_Create_DuplicateSlug verifies the unique slug constraint is enforced.
func TestRepository_Create_DuplicateSlug(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	repo := provider.NewRepository(pool)
	ctx := context.Background()

	p1 := newTestProvider("dup-slug")
	require.NoError(t, repo.Create(ctx, p1))

	p2 := newTestProvider("dup-slug")
	p2.Email = "other@example.com" // different email, same slug
	err := repo.Create(ctx, p2)
	require.Error(t, err, "creating a provider with a duplicate slug must fail")
}

// TestRepository_GetByID verifies retrieval of an existing provider and the not-found path.
func TestRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	repo := provider.NewRepository(pool)
	ctx := context.Background()

	p := newTestProvider("get-by-id")
	require.NoError(t, repo.Create(ctx, p))

	t.Run("existing", func(t *testing.T) {
		got, err := repo.GetByID(ctx, p.ID)
		require.NoError(t, err)
		assert.Equal(t, p.ID, got.ID)
		assert.Equal(t, p.Email, got.Email)
		assert.Equal(t, p.Slug, got.Slug)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestRepository_GetByEmail verifies retrieval by email and the not-found path.
func TestRepository_GetByEmail(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	repo := provider.NewRepository(pool)
	ctx := context.Background()

	p := newTestProvider("get-by-email")
	require.NoError(t, repo.Create(ctx, p))

	t.Run("existing", func(t *testing.T) {
		got, err := repo.GetByEmail(ctx, p.Email)
		require.NoError(t, err)
		assert.Equal(t, p.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetByEmail(ctx, "nobody@nowhere.com")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestRepository_GetBySlug verifies retrieval by slug and the not-found path.
func TestRepository_GetBySlug(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	repo := provider.NewRepository(pool)
	ctx := context.Background()

	p := newTestProvider("get-by-slug")
	require.NoError(t, repo.Create(ctx, p))

	t.Run("existing", func(t *testing.T) {
		got, err := repo.GetBySlug(ctx, p.Slug)
		require.NoError(t, err)
		assert.Equal(t, p.ID, got.ID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetBySlug(ctx, "does-not-exist")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestRepository_GetByGoogleID verifies that a provider created with a google_id can be retrieved.
func TestRepository_GetByGoogleID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	repo := provider.NewRepository(pool)
	ctx := context.Background()

	googleID := "google-oauth2|123456789"
	p := newTestProvider("get-by-google-id")
	p.GoogleID = &googleID
	require.NoError(t, repo.Create(ctx, p))

	t.Run("existing", func(t *testing.T) {
		got, err := repo.GetByGoogleID(ctx, googleID)
		require.NoError(t, err)
		assert.Equal(t, p.ID, got.ID)
		require.NotNil(t, got.GoogleID)
		assert.Equal(t, googleID, *got.GoogleID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetByGoogleID(ctx, "nonexistent-google-id")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestRepository_LinkGoogleID verifies that a google_id can be linked to an existing provider.
func TestRepository_LinkGoogleID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	repo := provider.NewRepository(pool)
	ctx := context.Background()

	p := newTestProvider("link-google-id")
	require.NoError(t, repo.Create(ctx, p))

	googleID := "google-oauth2|linked-999"

	t.Run("links successfully", func(t *testing.T) {
		err := repo.LinkGoogleID(ctx, p.ID, googleID)
		require.NoError(t, err)

		got, err := repo.GetByGoogleID(ctx, googleID)
		require.NoError(t, err)
		assert.Equal(t, p.ID, got.ID)
	})

	t.Run("not found provider", func(t *testing.T) {
		err := repo.LinkGoogleID(ctx, uuid.New(), "some-google-id")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestRepository_Update verifies that provider profile fields are updated correctly.
func TestRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	repo := provider.NewRepository(pool)
	ctx := context.Background()

	p := newTestProvider("update-provider")
	require.NoError(t, repo.Create(ctx, p))

	originalUpdatedAt := p.UpdatedAt

	// Ensure at least one millisecond passes so updated_at will differ.
	time.Sleep(2 * time.Millisecond)

	address := "Av. Corrientes 1234"
	businessName := "ACME Corp"
	p.Name = "Updated Name"
	p.Phone = "+5491100000011"
	p.Address = &address
	p.Timezone = "UTC"
	p.BusinessName = &businessName

	t.Run("updates fields", func(t *testing.T) {
		err := repo.Update(ctx, p)
		require.NoError(t, err)
		assert.True(t, p.UpdatedAt.After(originalUpdatedAt), "updated_at should be bumped")

		got, err := repo.GetByID(ctx, p.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", got.Name)
		assert.Equal(t, "+5491100000011", got.Phone)
		require.NotNil(t, got.Address)
		assert.Equal(t, address, *got.Address)
		assert.Equal(t, "UTC", got.Timezone)
		require.NotNil(t, got.BusinessName)
		assert.Equal(t, businessName, *got.BusinessName)
	})

	t.Run("not found", func(t *testing.T) {
		ghost := newTestProvider("ghost-slug")
		ghost.ID = uuid.New()
		err := repo.Update(ctx, ghost)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// ---------------------------------------------------------------------------
// Service repository tests
// ---------------------------------------------------------------------------

// newTestService returns a Service ready for insertion under the given provider.
func newTestService(providerID uuid.UUID, name string) *provider.Service {
	return &provider.Service{
		ProviderID:      providerID,
		Name:            name,
		DurationMinutes: 30,
	}
}

// TestRepository_CreateService verifies that a service is inserted with correct defaults.
func TestRepository_CreateService(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	repo := provider.NewRepository(pool)
	ctx := context.Background()

	p := newTestProvider("create-service-provider")
	require.NoError(t, repo.Create(ctx, p))

	svc := newTestService(p.ID, "Haircut")
	err := repo.CreateService(ctx, svc)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, svc.ID, "ID should be set after create")
	assert.True(t, svc.IsActive, "IsActive should default to true")
	assert.False(t, svc.CreatedAt.IsZero(), "CreatedAt should be set")
	assert.False(t, svc.UpdatedAt.IsZero(), "UpdatedAt should be set")
}

// TestRepository_GetServiceByID verifies retrieval by ID and the not-found path.
func TestRepository_GetServiceByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	repo := provider.NewRepository(pool)
	ctx := context.Background()

	p := newTestProvider("get-service-by-id-provider")
	require.NoError(t, repo.Create(ctx, p))
	svc := newTestService(p.ID, "Massage")
	require.NoError(t, repo.CreateService(ctx, svc))

	t.Run("existing", func(t *testing.T) {
		got, err := repo.GetServiceByID(ctx, svc.ID)
		require.NoError(t, err)
		assert.Equal(t, svc.ID, got.ID)
		assert.Equal(t, svc.Name, got.Name)
		assert.Equal(t, p.ID, got.ProviderID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetServiceByID(ctx, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestRepository_ListServices verifies that services are returned in created_at ASC order.
func TestRepository_ListServices(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	repo := provider.NewRepository(pool)
	ctx := context.Background()

	p := newTestProvider("list-services-provider")
	require.NoError(t, repo.Create(ctx, p))

	names := []string{"Alpha", "Beta", "Gamma"}
	for _, name := range names {
		svc := newTestService(p.ID, name)
		require.NoError(t, repo.CreateService(ctx, svc))
		// Small pause to guarantee distinct created_at values.
		time.Sleep(2 * time.Millisecond)
	}

	services, err := repo.ListServices(ctx, p.ID)
	require.NoError(t, err)
	require.Len(t, services, len(names), "should return all created services")

	for i, name := range names {
		assert.Equal(t, name, services[i].Name, "services should be ordered ASC by created_at")
	}
}

// TestRepository_ListServices_Empty verifies an empty slice is returned when there are no services.
func TestRepository_ListServices_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	repo := provider.NewRepository(pool)
	ctx := context.Background()

	p := newTestProvider("list-services-empty-provider")
	require.NoError(t, repo.Create(ctx, p))

	services, err := repo.ListServices(ctx, p.ID)
	require.NoError(t, err)
	assert.Empty(t, services)
}

// TestRepository_UpdateService verifies field updates and the not-found path.
func TestRepository_UpdateService(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	repo := provider.NewRepository(pool)
	ctx := context.Background()

	p := newTestProvider("update-service-provider")
	require.NoError(t, repo.Create(ctx, p))
	svc := newTestService(p.ID, "Original Name")
	require.NoError(t, repo.CreateService(ctx, svc))

	time.Sleep(2 * time.Millisecond)

	desc := "A great service"
	svc.Name = "Updated Name"
	svc.Description = &desc
	svc.DurationMinutes = 60
	svc.IsActive = false

	t.Run("updates fields", func(t *testing.T) {
		err := repo.UpdateService(ctx, svc)
		require.NoError(t, err)

		got, err := repo.GetServiceByID(ctx, svc.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", got.Name)
		require.NotNil(t, got.Description)
		assert.Equal(t, desc, *got.Description)
		assert.Equal(t, int16(60), got.DurationMinutes)
		assert.False(t, got.IsActive)
	})

	t.Run("not found", func(t *testing.T) {
		ghost := &provider.Service{ID: uuid.New(), Name: "Ghost", DurationMinutes: 10}
		err := repo.UpdateService(ctx, ghost)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestRepository_DeleteService verifies deletion and the not-found path.
func TestRepository_DeleteService(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	repo := provider.NewRepository(pool)
	ctx := context.Background()

	p := newTestProvider("delete-service-provider")
	require.NoError(t, repo.Create(ctx, p))
	svc := newTestService(p.ID, "To Delete")
	require.NoError(t, repo.CreateService(ctx, svc))

	t.Run("deletes existing", func(t *testing.T) {
		err := repo.DeleteService(ctx, svc.ID)
		require.NoError(t, err)

		_, err = repo.GetServiceByID(ctx, svc.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("not found", func(t *testing.T) {
		err := repo.DeleteService(ctx, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}
