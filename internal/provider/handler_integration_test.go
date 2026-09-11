package provider_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/THEcanon001/turnero/internal/platform/middleware"
	"github.com/THEcanon001/turnero/internal/provider"
	"github.com/THEcanon001/turnero/internal/testutil"
)

// ---------------------------------------------------------------------------
// Mock stats provider
// ---------------------------------------------------------------------------

type mockStats struct{}

func (m *mockStats) GetStats(_ context.Context, _ uuid.UUID, _, _ string) (*provider.StatsResult, error) {
	return &provider.StatsResult{
		Total:     10,
		Confirmed: 5,
		Completed: 3,
		Cancelled: 1,
		NoShow:    1,
	}, nil
}

func (m *mockStats) GetStatsByEmployee(_ context.Context, _ uuid.UUID, _, _ string) ([]provider.EmployeeStatsResult, error) {
	return []provider.EmployeeStatsResult{}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// seedProviderViaRepo creates a provider using the repository layer and returns it.
func seedProviderViaRepo(t *testing.T, repo *provider.Repository, slug string) *provider.Provider {
	t.Helper()
	p := &provider.Provider{
		Type:         provider.TypeIndividual,
		Name:         "Integration Provider",
		Slug:         slug,
		Phone:        "+5491100000099",
		Timezone:     "America/Argentina/Buenos_Aires",
		Email:        slug + "@integration.com",
		PasswordHash: "$2a$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ012",
	}
	err := repo.Create(context.Background(), p)
	require.NoError(t, err, "seedProviderViaRepo: create")
	return p
}

// withChiParam attaches a chi route context with the given key/value URL param.
func withChiParam(ctx context.Context, key, value string) context.Context {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return context.WithValue(ctx, chi.RouteCtxKey, rctx)
}

// jsonBody encodes v as JSON and returns a *bytes.Reader.
func jsonBody(t *testing.T, v any) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewReader(b)
}

// decodeJSON decodes the response body into out.
func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, out any) {
	t.Helper()
	require.NoError(t, json.NewDecoder(rec.Body).Decode(out))
}

// ---------------------------------------------------------------------------
// GetMe
// ---------------------------------------------------------------------------

func TestHandler_GetMe(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	repo := provider.NewRepository(pool)
	h := provider.NewHandler(repo)

	p := seedProviderViaRepo(t, repo, "get-me-handler")

	t.Run("returns provider JSON", func(t *testing.T) {
		ctx := middleware.WithTestAuth(context.Background(), p.ID, p.ID, "admin")
		req := httptest.NewRequest(http.MethodGet, "/v1/provider/me", nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.GetMe(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var got map[string]any
		decodeJSON(t, rec, &got)
		assert.Equal(t, p.ID.String(), got["id"])
		assert.Equal(t, p.Email, got["email"])
		assert.Equal(t, p.Slug, got["slug"])
		assert.Equal(t, string(provider.PlanFree), got["plan"])
		assert.Equal(t, true, got["is_active"])
	})

	t.Run("not found returns 404", func(t *testing.T) {
		ctx := middleware.WithTestAuth(context.Background(), uuid.New(), uuid.New(), "admin")
		req := httptest.NewRequest(http.MethodGet, "/v1/provider/me", nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.GetMe(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// UpdateMe
// ---------------------------------------------------------------------------

func TestHandler_UpdateMe(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	repo := provider.NewRepository(pool)
	h := provider.NewHandler(repo)

	p := seedProviderViaRepo(t, repo, "update-me-handler")

	t.Run("updates and returns provider", func(t *testing.T) {
		body := map[string]any{
			"name":     "New Name",
			"phone":    "+5491100000077",
			"timezone": "UTC",
		}
		ctx := middleware.WithTestAuth(context.Background(), p.ID, p.ID, "admin")
		req := httptest.NewRequest(http.MethodPut, "/v1/provider/me", jsonBody(t, body)).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.UpdateMe(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var got map[string]any
		decodeJSON(t, rec, &got)
		assert.Equal(t, "New Name", got["name"])
		assert.Equal(t, "+5491100000077", got["phone"])
		assert.Equal(t, "UTC", got["timezone"])
	})

	t.Run("not found returns 404", func(t *testing.T) {
		body := map[string]any{
			"name":     "Ghost",
			"phone":    "+5491100000000",
			"timezone": "UTC",
		}
		ctx := middleware.WithTestAuth(context.Background(), uuid.New(), uuid.New(), "admin")
		req := httptest.NewRequest(http.MethodPut, "/v1/provider/me", jsonBody(t, body)).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.UpdateMe(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// CreateService
// ---------------------------------------------------------------------------

func TestHandler_CreateService(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	repo := provider.NewRepository(pool)
	h := provider.NewHandler(repo)

	p := seedProviderViaRepo(t, repo, "create-service-handler")

	t.Run("creates service and returns 201", func(t *testing.T) {
		body := map[string]any{
			"name":             "Haircut",
			"duration_minutes": 30,
		}
		ctx := middleware.WithTestAuth(context.Background(), p.ID, p.ID, "admin")
		req := httptest.NewRequest(http.MethodPost, "/v1/services", jsonBody(t, body)).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.CreateService(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)

		var got map[string]any
		decodeJSON(t, rec, &got)
		assert.NotEmpty(t, got["id"])
		assert.Equal(t, "Haircut", got["name"])
		assert.Equal(t, float64(30), got["duration_minutes"])
		assert.Equal(t, true, got["is_active"])
	})
}

// ---------------------------------------------------------------------------
// ListServices
// ---------------------------------------------------------------------------

func TestHandler_ListServices(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	repo := provider.NewRepository(pool)
	h := provider.NewHandler(repo)

	p := seedProviderViaRepo(t, repo, "list-services-handler")

	// Seed two services.
	svc1 := &provider.Service{ProviderID: p.ID, Name: "Service A", DurationMinutes: 20}
	svc2 := &provider.Service{ProviderID: p.ID, Name: "Service B", DurationMinutes: 40}
	require.NoError(t, repo.CreateService(context.Background(), svc1))
	require.NoError(t, repo.CreateService(context.Background(), svc2))

	t.Run("returns array of services", func(t *testing.T) {
		ctx := middleware.WithTestAuth(context.Background(), p.ID, p.ID, "admin")
		req := httptest.NewRequest(http.MethodGet, "/v1/services", nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.ListServices(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var got []map[string]any
		decodeJSON(t, rec, &got)
		assert.Len(t, got, 2)
	})

	t.Run("returns empty array when provider has no services", func(t *testing.T) {
		empty := seedProviderViaRepo(t, repo, "list-services-empty-handler")
		ctx := middleware.WithTestAuth(context.Background(), empty.ID, empty.ID, "admin")
		req := httptest.NewRequest(http.MethodGet, "/v1/services", nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.ListServices(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var got []map[string]any
		decodeJSON(t, rec, &got)
		assert.Empty(t, got)
	})
}

// ---------------------------------------------------------------------------
// GetService
// ---------------------------------------------------------------------------

func TestHandler_GetService(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	repo := provider.NewRepository(pool)
	h := provider.NewHandler(repo)

	p := seedProviderViaRepo(t, repo, "get-service-handler")
	other := seedProviderViaRepo(t, repo, "get-service-other-handler")

	svc := &provider.Service{ProviderID: p.ID, Name: "Manicure", DurationMinutes: 45}
	require.NoError(t, repo.CreateService(context.Background(), svc))

	t.Run("returns service for owner", func(t *testing.T) {
		ctx := middleware.WithTestAuth(context.Background(), p.ID, p.ID, "admin")
		ctx = withChiParam(ctx, "id", svc.ID.String())
		req := httptest.NewRequest(http.MethodGet, "/v1/services/"+svc.ID.String(), nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.GetService(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var got map[string]any
		decodeJSON(t, rec, &got)
		assert.Equal(t, svc.ID.String(), got["id"])
	})

	t.Run("not found for random UUID", func(t *testing.T) {
		ctx := middleware.WithTestAuth(context.Background(), p.ID, p.ID, "admin")
		nonexistent := uuid.New()
		ctx = withChiParam(ctx, "id", nonexistent.String())
		req := httptest.NewRequest(http.MethodGet, "/v1/services/"+nonexistent.String(), nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.GetService(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("forbidden for different provider", func(t *testing.T) {
		ctx := middleware.WithTestAuth(context.Background(), other.ID, other.ID, "admin")
		ctx = withChiParam(ctx, "id", svc.ID.String())
		req := httptest.NewRequest(http.MethodGet, "/v1/services/"+svc.ID.String(), nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.GetService(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// UpdateService
// ---------------------------------------------------------------------------

func TestHandler_UpdateService(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	repo := provider.NewRepository(pool)
	h := provider.NewHandler(repo)

	p := seedProviderViaRepo(t, repo, "update-service-handler")
	other := seedProviderViaRepo(t, repo, "update-service-other-handler")

	svc := &provider.Service{ProviderID: p.ID, Name: "Pedicure", DurationMinutes: 50}
	require.NoError(t, repo.CreateService(context.Background(), svc))

	updateBody := map[string]any{
		"name":             "Pedicure Deluxe",
		"duration_minutes": 60,
		"is_active":        true,
	}

	t.Run("updates service for owner", func(t *testing.T) {
		ctx := middleware.WithTestAuth(context.Background(), p.ID, p.ID, "admin")
		ctx = withChiParam(ctx, "id", svc.ID.String())
		req := httptest.NewRequest(http.MethodPut, "/v1/services/"+svc.ID.String(), jsonBody(t, updateBody)).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.UpdateService(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var got map[string]any
		decodeJSON(t, rec, &got)
		assert.Equal(t, "Pedicure Deluxe", got["name"])
		assert.Equal(t, float64(60), got["duration_minutes"])
	})

	t.Run("not found returns 404", func(t *testing.T) {
		nonexistent := uuid.New()
		ctx := middleware.WithTestAuth(context.Background(), p.ID, p.ID, "admin")
		ctx = withChiParam(ctx, "id", nonexistent.String())
		req := httptest.NewRequest(http.MethodPut, "/v1/services/"+nonexistent.String(), jsonBody(t, updateBody)).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.UpdateService(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("forbidden for different provider", func(t *testing.T) {
		ctx := middleware.WithTestAuth(context.Background(), other.ID, other.ID, "admin")
		ctx = withChiParam(ctx, "id", svc.ID.String())
		req := httptest.NewRequest(http.MethodPut, "/v1/services/"+svc.ID.String(), jsonBody(t, updateBody)).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.UpdateService(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// DeleteService
// ---------------------------------------------------------------------------

func TestHandler_DeleteService(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	repo := provider.NewRepository(pool)
	h := provider.NewHandler(repo)

	p := seedProviderViaRepo(t, repo, "delete-service-handler")
	other := seedProviderViaRepo(t, repo, "delete-service-other-handler")

	t.Run("deletes service and returns 204", func(t *testing.T) {
		svc := &provider.Service{ProviderID: p.ID, Name: "Waxing", DurationMinutes: 35}
		require.NoError(t, repo.CreateService(context.Background(), svc))

		ctx := middleware.WithTestAuth(context.Background(), p.ID, p.ID, "admin")
		ctx = withChiParam(ctx, "id", svc.ID.String())
		req := httptest.NewRequest(http.MethodDelete, "/v1/services/"+svc.ID.String(), nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.DeleteService(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)

		// Verify it is actually gone.
		_, err := repo.GetServiceByID(context.Background(), svc.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("not found returns 404", func(t *testing.T) {
		nonexistent := uuid.New()
		ctx := middleware.WithTestAuth(context.Background(), p.ID, p.ID, "admin")
		ctx = withChiParam(ctx, "id", nonexistent.String())
		req := httptest.NewRequest(http.MethodDelete, "/v1/services/"+nonexistent.String(), nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.DeleteService(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("forbidden for different provider", func(t *testing.T) {
		// Create service under p, then attempt delete as other.
		svc := &provider.Service{ProviderID: p.ID, Name: "Facials", DurationMinutes: 30}
		require.NoError(t, repo.CreateService(context.Background(), svc))

		ctx := middleware.WithTestAuth(context.Background(), other.ID, other.ID, "admin")
		ctx = withChiParam(ctx, "id", svc.ID.String())
		req := httptest.NewRequest(http.MethodDelete, "/v1/services/"+svc.ID.String(), nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.DeleteService(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

// ---------------------------------------------------------------------------
// GetStats
// ---------------------------------------------------------------------------

func TestHandler_GetStats(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	repo := provider.NewRepository(pool)

	p := seedProviderViaRepo(t, repo, "get-stats-handler")

	t.Run("returns stats with valid params", func(t *testing.T) {
		h := provider.NewHandler(repo)
		h.SetStatsProvider(&mockStats{})

		ctx := middleware.WithTestAuth(context.Background(), p.ID, p.ID, "admin")
		req := httptest.NewRequest(http.MethodGet, "/v1/provider/me/stats?from=2026-01-01&to=2026-01-31", nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.GetStats(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var got map[string]any
		decodeJSON(t, rec, &got)
		require.Contains(t, got, "summary")
		require.Contains(t, got, "by_employee")

		summary, ok := got["summary"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(10), summary["total"])
		assert.Equal(t, float64(5), summary["confirmed"])
		assert.Equal(t, float64(3), summary["completed"])
		assert.Equal(t, float64(1), summary["cancelled"])
		assert.Equal(t, float64(1), summary["no_show"])
	})

	t.Run("missing from param returns 400", func(t *testing.T) {
		h := provider.NewHandler(repo)
		h.SetStatsProvider(&mockStats{})

		ctx := middleware.WithTestAuth(context.Background(), p.ID, p.ID, "admin")
		req := httptest.NewRequest(http.MethodGet, "/v1/provider/me/stats?to=2026-01-31", nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.GetStats(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("missing to param returns 400", func(t *testing.T) {
		h := provider.NewHandler(repo)
		h.SetStatsProvider(&mockStats{})

		ctx := middleware.WithTestAuth(context.Background(), p.ID, p.ID, "admin")
		req := httptest.NewRequest(http.MethodGet, "/v1/provider/me/stats?from=2026-01-01", nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.GetStats(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("missing both params returns 400", func(t *testing.T) {
		h := provider.NewHandler(repo)
		h.SetStatsProvider(&mockStats{})

		ctx := middleware.WithTestAuth(context.Background(), p.ID, p.ID, "admin")
		req := httptest.NewRequest(http.MethodGet, "/v1/provider/me/stats", nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.GetStats(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("nil statsProvider returns 500", func(t *testing.T) {
		h := provider.NewHandler(repo) // statsProvider is nil by default

		ctx := middleware.WithTestAuth(context.Background(), p.ID, p.ID, "admin")
		req := httptest.NewRequest(http.MethodGet, "/v1/provider/me/stats?from=2026-01-01&to=2026-01-31", nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.GetStats(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}
