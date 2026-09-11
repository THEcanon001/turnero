package employee_test

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

	"github.com/THEcanon001/turnero/internal/employee"
	"github.com/THEcanon001/turnero/internal/platform/middleware"
	"github.com/THEcanon001/turnero/internal/testutil"
)

// mockCanceller is a test double for the AppointmentCanceller interface.
type mockCanceller struct {
	called bool
	count  int64
	err    error
}

func (m *mockCanceller) CancelByEmployeeAndDateRange(ctx context.Context, employeeID uuid.UUID, fromDate, toDate, reason string) (int64, error) {
	m.called = true
	return m.count, m.err
}

// withChiParam attaches a chi route context with a single URL parameter to the given request context.
func withChiParam(ctx context.Context, key, value string) context.Context {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return context.WithValue(ctx, chi.RouteCtxKey, rctx)
}

// TestHandler_List verifies that the List handler returns all employees for a provider.
func TestHandler_List(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "hdl-list")
	testutil.SeedEmployee(t, pool, providerID, "Alice")
	testutil.SeedEmployee(t, pool, providerID, "Bob")

	h := employee.NewHandler(employee.NewRepository(pool))

	req := httptest.NewRequest(http.MethodGet, "/v1/employees", nil)
	ctx := middleware.WithTestAuth(req.Context(), uuid.New(), uuid.MustParse(providerID), "admin")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.List(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var got []map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Len(t, got, 2)
}

// TestHandler_List_Empty verifies that the List handler returns an empty array when there are
// no employees for the provider.
func TestHandler_List_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "hdl-list-empty")
	h := employee.NewHandler(employee.NewRepository(pool))

	req := httptest.NewRequest(http.MethodGet, "/v1/employees", nil)
	ctx := middleware.WithTestAuth(req.Context(), uuid.New(), uuid.MustParse(providerID), "admin")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.List(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var got []map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Empty(t, got)
}

// TestHandler_GetByID verifies that the handler returns the employee and their service IDs.
func TestHandler_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "hdl-getbyid")
	empID := testutil.SeedEmployee(t, pool, providerID, "Target Employee")
	svcID := testutil.SeedService(t, pool, providerID, "Haircut", 30)

	repo := employee.NewRepository(pool)
	require.NoError(t, repo.AssignServices(context.Background(), uuid.MustParse(empID), []uuid.UUID{uuid.MustParse(svcID)}))

	h := employee.NewHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/v1/employees/"+empID, nil)
	ctx := middleware.WithTestAuth(req.Context(), uuid.New(), uuid.MustParse(providerID), "admin")
	ctx = withChiParam(ctx, "id", empID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetByID(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Contains(t, resp, "employee")
	assert.Contains(t, resp, "service_ids")

	serviceIDs, ok := resp["service_ids"].([]any)
	require.True(t, ok)
	assert.Len(t, serviceIDs, 1)
}

// TestHandler_GetByID_NotFound verifies that a 404 is returned for a missing employee.
func TestHandler_GetByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "hdl-getbyid-nf")
	h := employee.NewHandler(employee.NewRepository(pool))

	missingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/v1/employees/"+missingID, nil)
	ctx := middleware.WithTestAuth(req.Context(), uuid.New(), uuid.MustParse(providerID), "admin")
	ctx = withChiParam(ctx, "id", missingID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetByID(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// TestHandler_GetByID_Forbidden verifies that accessing another provider's employee returns 403.
func TestHandler_GetByID_Forbidden(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ownerProviderID := testutil.SeedProvider(t, pool, "hdl-getbyid-owner")
	otherProviderID := testutil.SeedProvider(t, pool, "hdl-getbyid-other")
	empID := testutil.SeedEmployee(t, pool, ownerProviderID, "Owned Employee")

	h := employee.NewHandler(employee.NewRepository(pool))

	req := httptest.NewRequest(http.MethodGet, "/v1/employees/"+empID, nil)
	ctx := middleware.WithTestAuth(req.Context(), uuid.New(), uuid.MustParse(otherProviderID), "admin")
	ctx = withChiParam(ctx, "id", empID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.GetByID(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// TestHandler_Create verifies that POSTing a valid body creates the employee and returns 201.
func TestHandler_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "hdl-create")
	h := employee.NewHandler(employee.NewRepository(pool))

	body, _ := json.Marshal(map[string]string{
		"name":  "New Employee",
		"phone": "+5491100000011",
		"role":  "employee",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/employees", bytes.NewReader(body))
	ctx := middleware.WithTestAuth(req.Context(), uuid.New(), uuid.MustParse(providerID), "admin")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var got map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, "New Employee", got["name"])
	assert.Equal(t, "+5491100000011", got["phone"])
	assert.NotEmpty(t, got["id"])
}

// TestHandler_Create_Duplicate verifies that creating an employee with a duplicate phone
// returns 409.
func TestHandler_Create_Duplicate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "hdl-create-dup")
	testutil.SeedEmployee(t, pool, providerID, "Existing Employee")

	// SeedEmployee uses +5491100000001, try to create another with the same phone
	h := employee.NewHandler(employee.NewRepository(pool))

	body, _ := json.Marshal(map[string]string{
		"name":  "Another Employee",
		"phone": "+5491100000001",
		"role":  "employee",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/employees", bytes.NewReader(body))
	ctx := middleware.WithTestAuth(req.Context(), uuid.New(), uuid.MustParse(providerID), "admin")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

// TestHandler_Update verifies that PUTting valid data updates the employee and returns 200.
func TestHandler_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "hdl-update")
	empID := testutil.SeedEmployee(t, pool, providerID, "Before Update")

	h := employee.NewHandler(employee.NewRepository(pool))

	isActive := true
	body, _ := json.Marshal(map[string]any{
		"name":      "After Update",
		"phone":     "+5491100000022",
		"is_active": isActive,
	})
	req := httptest.NewRequest(http.MethodPut, "/v1/employees/"+empID, bytes.NewReader(body))
	ctx := middleware.WithTestAuth(req.Context(), uuid.New(), uuid.MustParse(providerID), "admin")
	ctx = withChiParam(ctx, "id", empID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.Update(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Contains(t, resp, "employee")
}

// TestHandler_Update_Deactivation verifies that deactivating an employee triggers the
// appointment canceller.
func TestHandler_Update_Deactivation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "hdl-update-deact")
	empID := testutil.SeedEmployee(t, pool, providerID, "Active Employee")

	h := employee.NewHandler(employee.NewRepository(pool))
	mock := &mockCanceller{count: 3}
	h.SetAppointmentCanceller(mock)

	isActive := false
	body, _ := json.Marshal(map[string]any{
		"name":      "Active Employee",
		"phone":     "+5491100000001",
		"is_active": isActive,
	})
	req := httptest.NewRequest(http.MethodPut, "/v1/employees/"+empID, bytes.NewReader(body))
	ctx := middleware.WithTestAuth(req.Context(), uuid.New(), uuid.MustParse(providerID), "admin")
	ctx = withChiParam(ctx, "id", empID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.Update(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, mock.called, "appointment canceller should have been called on deactivation")

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	// cancelled_appointments should reflect mock return value
	cancelled, ok := resp["cancelled_appointments"].(float64)
	require.True(t, ok)
	assert.Equal(t, float64(3), cancelled)
}

// TestHandler_Update_Deactivation_NoCancellerSet verifies that deactivating with no
// canceller set does not panic and still returns 200.
func TestHandler_Update_Deactivation_NoCancellerSet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "hdl-update-no-cancel")
	empID := testutil.SeedEmployee(t, pool, providerID, "Active Employee NC")

	// No SetAppointmentCanceller call
	h := employee.NewHandler(employee.NewRepository(pool))

	isActive := false
	body, _ := json.Marshal(map[string]any{
		"name":      "Active Employee NC",
		"phone":     "+5491100000001",
		"is_active": isActive,
	})
	req := httptest.NewRequest(http.MethodPut, "/v1/employees/"+empID, bytes.NewReader(body))
	ctx := middleware.WithTestAuth(req.Context(), uuid.New(), uuid.MustParse(providerID), "admin")
	ctx = withChiParam(ctx, "id", empID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.Update(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestHandler_Update_NotFound verifies that updating a non-existent employee returns 404.
func TestHandler_Update_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "hdl-update-nf")
	h := employee.NewHandler(employee.NewRepository(pool))

	missingID := uuid.New().String()
	body, _ := json.Marshal(map[string]any{
		"name":  "Ghost",
		"phone": "+5491100000033",
	})
	req := httptest.NewRequest(http.MethodPut, "/v1/employees/"+missingID, bytes.NewReader(body))
	ctx := middleware.WithTestAuth(req.Context(), uuid.New(), uuid.MustParse(providerID), "admin")
	ctx = withChiParam(ctx, "id", missingID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.Update(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// TestHandler_Update_Forbidden verifies that updating another provider's employee returns 403.
func TestHandler_Update_Forbidden(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ownerProviderID := testutil.SeedProvider(t, pool, "hdl-update-owner")
	otherProviderID := testutil.SeedProvider(t, pool, "hdl-update-other")
	empID := testutil.SeedEmployee(t, pool, ownerProviderID, "Owned Employee")

	h := employee.NewHandler(employee.NewRepository(pool))

	body, _ := json.Marshal(map[string]any{
		"name":  "Owned Employee",
		"phone": "+5491100000001",
	})
	req := httptest.NewRequest(http.MethodPut, "/v1/employees/"+empID, bytes.NewReader(body))
	ctx := middleware.WithTestAuth(req.Context(), uuid.New(), uuid.MustParse(otherProviderID), "admin")
	ctx = withChiParam(ctx, "id", empID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.Update(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// TestHandler_Delete verifies that deleting an employee returns 204.
func TestHandler_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "hdl-delete")
	empID := testutil.SeedEmployee(t, pool, providerID, "To Be Deleted")

	h := employee.NewHandler(employee.NewRepository(pool))

	req := httptest.NewRequest(http.MethodDelete, "/v1/employees/"+empID, nil)
	ctx := middleware.WithTestAuth(req.Context(), uuid.New(), uuid.MustParse(providerID), "admin")
	ctx = withChiParam(ctx, "id", empID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

// TestHandler_Delete_NotFound verifies that deleting a non-existent employee returns 404.
func TestHandler_Delete_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "hdl-delete-nf")
	h := employee.NewHandler(employee.NewRepository(pool))

	missingID := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/v1/employees/"+missingID, nil)
	ctx := middleware.WithTestAuth(req.Context(), uuid.New(), uuid.MustParse(providerID), "admin")
	ctx = withChiParam(ctx, "id", missingID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// TestHandler_Delete_Forbidden verifies that deleting another provider's employee returns 403.
func TestHandler_Delete_Forbidden(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ownerProviderID := testutil.SeedProvider(t, pool, "hdl-delete-owner")
	otherProviderID := testutil.SeedProvider(t, pool, "hdl-delete-other")
	empID := testutil.SeedEmployee(t, pool, ownerProviderID, "Owned Employee")

	h := employee.NewHandler(employee.NewRepository(pool))

	req := httptest.NewRequest(http.MethodDelete, "/v1/employees/"+empID, nil)
	ctx := middleware.WithTestAuth(req.Context(), uuid.New(), uuid.MustParse(otherProviderID), "admin")
	ctx = withChiParam(ctx, "id", empID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// TestHandler_CreateInvitation verifies that POSTing a valid body creates an invitation
// and returns 201 with a code.
func TestHandler_CreateInvitation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "hdl-create-inv")
	h := employee.NewHandler(employee.NewRepository(pool))

	body, _ := json.Marshal(map[string]string{
		"employee_name": "Invited Person",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/employees/invitations", bytes.NewReader(body))
	ctx := middleware.WithTestAuth(req.Context(), uuid.New(), uuid.MustParse(providerID), "admin")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.CreateInvitation(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var got map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.NotEmpty(t, got["code"])
	assert.Equal(t, "Invited Person", got["employee_name"])
	assert.NotEmpty(t, got["id"])
}

// TestHandler_ListInvitations verifies that the handler returns all invitations for a provider.
func TestHandler_ListInvitations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "hdl-list-inv")

	// Pre-seed two invitations directly via repository
	repo := employee.NewRepository(pool)
	ctx := context.Background()
	_, err := repo.CreateInvitation(ctx, uuid.MustParse(providerID), "Invitee One", 48*1000000000)
	require.NoError(t, err)
	_, err = repo.CreateInvitation(ctx, uuid.MustParse(providerID), "Invitee Two", 48*1000000000)
	require.NoError(t, err)

	h := employee.NewHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/v1/employees/invitations", nil)
	authCtx := middleware.WithTestAuth(req.Context(), uuid.New(), uuid.MustParse(providerID), "admin")
	req = req.WithContext(authCtx)
	rec := httptest.NewRecorder()

	h.ListInvitations(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var got []map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Len(t, got, 2)
}

// TestHandler_ListInvitations_Empty verifies that an empty array is returned when there
// are no invitations.
func TestHandler_ListInvitations_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "hdl-list-inv-empty")
	h := employee.NewHandler(employee.NewRepository(pool))

	req := httptest.NewRequest(http.MethodGet, "/v1/employees/invitations", nil)
	ctx := middleware.WithTestAuth(req.Context(), uuid.New(), uuid.MustParse(providerID), "admin")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.ListInvitations(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var got []map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Empty(t, got)
}

// TestHandler_AssignServices verifies that PUTting service IDs assigns them and returns 200.
func TestHandler_AssignServices(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "hdl-assign-svc")
	empID := testutil.SeedEmployee(t, pool, providerID, "Employee with Services")
	svc1ID := testutil.SeedService(t, pool, providerID, "Massage", 60)
	svc2ID := testutil.SeedService(t, pool, providerID, "Facial", 45)

	h := employee.NewHandler(employee.NewRepository(pool))

	body, _ := json.Marshal(map[string]any{
		"service_ids": []string{svc1ID, svc2ID},
	})
	req := httptest.NewRequest(http.MethodPut, "/v1/employees/"+empID+"/services", bytes.NewReader(body))
	ctx := middleware.WithTestAuth(req.Context(), uuid.New(), uuid.MustParse(providerID), "admin")
	ctx = withChiParam(ctx, "id", empID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.AssignServices(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, empID, resp["employee_id"])

	serviceIDs, ok := resp["service_ids"].([]any)
	require.True(t, ok)
	assert.Len(t, serviceIDs, 2)
}

// TestHandler_AssignServices_NotFound verifies that assigning services to a non-existent
// employee returns 404.
func TestHandler_AssignServices_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "hdl-assign-svc-nf")
	svcID := testutil.SeedService(t, pool, providerID, "Service", 30)

	h := employee.NewHandler(employee.NewRepository(pool))

	missingID := uuid.New().String()
	body, _ := json.Marshal(map[string]any{
		"service_ids": []string{svcID},
	})
	req := httptest.NewRequest(http.MethodPut, "/v1/employees/"+missingID+"/services", bytes.NewReader(body))
	ctx := middleware.WithTestAuth(req.Context(), uuid.New(), uuid.MustParse(providerID), "admin")
	ctx = withChiParam(ctx, "id", missingID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.AssignServices(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// TestHandler_AssignServices_Forbidden verifies that assigning services to another provider's
// employee returns 403.
func TestHandler_AssignServices_Forbidden(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ownerProviderID := testutil.SeedProvider(t, pool, "hdl-assign-svc-owner")
	otherProviderID := testutil.SeedProvider(t, pool, "hdl-assign-svc-other")
	empID := testutil.SeedEmployee(t, pool, ownerProviderID, "Owned Employee")
	svcID := testutil.SeedService(t, pool, otherProviderID, "Service", 30)

	h := employee.NewHandler(employee.NewRepository(pool))

	body, _ := json.Marshal(map[string]any{
		"service_ids": []string{svcID},
	})
	req := httptest.NewRequest(http.MethodPut, "/v1/employees/"+empID+"/services", bytes.NewReader(body))
	ctx := middleware.WithTestAuth(req.Context(), uuid.New(), uuid.MustParse(otherProviderID), "admin")
	ctx = withChiParam(ctx, "id", empID)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.AssignServices(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}
