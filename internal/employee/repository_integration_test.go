package employee_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/THEcanon001/turnero/internal/employee"
	"github.com/THEcanon001/turnero/internal/testutil"
)

func strPtr(s string) *string { return &s }

// TestRepository_Create verifies employee creation sets ID, IsActive, and timestamps.
func TestRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "emp-repo-create")
	repo := employee.NewRepository(pool)
	ctx := context.Background()

	emp := &employee.Employee{
		ProviderID: uuid.MustParse(providerID),
		Name:       "Test Employee",
		Phone:      "+5491100000099",
		Role:       employee.RoleEmployee,
	}

	err := repo.Create(ctx, emp)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, emp.ID)
	assert.True(t, emp.IsActive)
	assert.False(t, emp.CreatedAt.IsZero())
	assert.False(t, emp.UpdatedAt.IsZero())
	assert.Equal(t, uuid.MustParse(providerID), emp.ProviderID)
	assert.Equal(t, "Test Employee", emp.Name)
	assert.Equal(t, employee.RoleEmployee, emp.Role)
}

// TestRepository_Create_WithEmail verifies admin employee creation with email and password hash.
func TestRepository_Create_WithEmail(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "emp-repo-email")
	repo := employee.NewRepository(pool)
	ctx := context.Background()

	email := "admin@test.com"
	emp := &employee.Employee{
		ProviderID:   uuid.MustParse(providerID),
		Name:         "Admin",
		Phone:        "+5491100000098",
		Role:         employee.RoleAdmin,
		Email:        &email,
		PasswordHash: strPtr("$2a$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ012"),
	}

	err := repo.Create(ctx, emp)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, emp.ID)
	require.NotNil(t, emp.Email)
	assert.Equal(t, "admin@test.com", *emp.Email)
}

// TestRepository_Create_DuplicatePhone verifies that inserting two employees with the same
// phone number under the same provider returns a constraint violation error.
func TestRepository_Create_DuplicatePhone(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "emp-repo-dup")
	repo := employee.NewRepository(pool)
	ctx := context.Background()

	first := &employee.Employee{
		ProviderID: uuid.MustParse(providerID),
		Name:       "First",
		Phone:      "+5491100000097",
		Role:       employee.RoleEmployee,
	}
	require.NoError(t, repo.Create(ctx, first))

	second := &employee.Employee{
		ProviderID: uuid.MustParse(providerID),
		Name:       "Second",
		Phone:      "+5491100000097", // same phone
		Role:       employee.RoleEmployee,
	}
	err := repo.Create(ctx, second)
	require.Error(t, err)
}

// TestRepository_GetByID verifies retrieval of an existing employee by ID.
func TestRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "emp-repo-getbyid")
	empID := testutil.SeedEmployee(t, pool, providerID, "Jane Doe")
	repo := employee.NewRepository(pool)
	ctx := context.Background()

	got, err := repo.GetByID(ctx, uuid.MustParse(empID))
	require.NoError(t, err)

	assert.Equal(t, uuid.MustParse(empID), got.ID)
	assert.Equal(t, "Jane Doe", got.Name)
	assert.Equal(t, uuid.MustParse(providerID), got.ProviderID)
	assert.True(t, got.IsActive)
}

// TestRepository_GetByID_NotFound verifies "not found" error for a missing ID.
func TestRepository_GetByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := employee.NewRepository(pool)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestRepository_ListByProvider verifies listing all employees for a given provider.
func TestRepository_ListByProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "emp-repo-list")
	testutil.SeedEmployee(t, pool, providerID, "Alice")
	testutil.SeedEmployee(t, pool, providerID, "Bob")
	repo := employee.NewRepository(pool)
	ctx := context.Background()

	employees, err := repo.ListByProvider(ctx, uuid.MustParse(providerID))
	require.NoError(t, err)

	assert.Len(t, employees, 2)
	names := []string{employees[0].Name, employees[1].Name}
	assert.Contains(t, names, "Alice")
	assert.Contains(t, names, "Bob")
}

// TestRepository_ListByProvider_Empty verifies an empty list for a provider with no employees.
func TestRepository_ListByProvider_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "emp-repo-list-empty")
	repo := employee.NewRepository(pool)
	ctx := context.Background()

	employees, err := repo.ListByProvider(ctx, uuid.MustParse(providerID))
	require.NoError(t, err)
	assert.Empty(t, employees)
}

// TestRepository_Update verifies that updating name, phone, and is_active works correctly
// and that updated_at is refreshed.
func TestRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "emp-repo-update")
	empID := testutil.SeedEmployee(t, pool, providerID, "Original Name")
	repo := employee.NewRepository(pool)
	ctx := context.Background()

	emp, err := repo.GetByID(ctx, uuid.MustParse(empID))
	require.NoError(t, err)

	originalUpdatedAt := emp.UpdatedAt
	// Small sleep to ensure updated_at changes measurably
	time.Sleep(10 * time.Millisecond)

	emp.Name = "Updated Name"
	emp.Phone = "+5491100000055"
	emp.IsActive = false

	err = repo.Update(ctx, emp)
	require.NoError(t, err)

	assert.Equal(t, "Updated Name", emp.Name)
	assert.Equal(t, "+5491100000055", emp.Phone)
	assert.False(t, emp.IsActive)
	assert.True(t, emp.UpdatedAt.After(originalUpdatedAt) || emp.UpdatedAt.Equal(originalUpdatedAt))

	// Confirm from DB
	updated, err := repo.GetByID(ctx, uuid.MustParse(empID))
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.False(t, updated.IsActive)
}

// TestRepository_Update_NotFound verifies "not found" error when updating a non-existent employee.
func TestRepository_Update_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := employee.NewRepository(pool)
	ctx := context.Background()

	ghost := &employee.Employee{
		ID:       uuid.New(),
		Name:     "Ghost",
		Phone:    "+5491100000000",
		IsActive: true,
	}
	err := repo.Update(ctx, ghost)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestRepository_Delete verifies hard deletion and that GetByID fails afterwards.
func TestRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "emp-repo-delete")
	empID := testutil.SeedEmployee(t, pool, providerID, "To Delete")
	repo := employee.NewRepository(pool)
	ctx := context.Background()

	err := repo.Delete(ctx, uuid.MustParse(empID))
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, uuid.MustParse(empID))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestRepository_Delete_NotFound verifies an error when deleting a non-existent employee.
func TestRepository_Delete_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := employee.NewRepository(pool)
	ctx := context.Background()

	err := repo.Delete(ctx, uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestRepository_GetByEmail verifies retrieval by email for admin employees.
func TestRepository_GetByEmail(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "emp-repo-byemail")
	testutil.SeedAdminEmployee(t, pool, providerID, "Admin User", "admin@example.com")
	repo := employee.NewRepository(pool)
	ctx := context.Background()

	got, err := repo.GetByEmail(ctx, "admin@example.com")
	require.NoError(t, err)
	require.NotNil(t, got.Email)
	assert.Equal(t, "admin@example.com", *got.Email)
	assert.Equal(t, employee.RoleAdmin, got.Role)
}

// TestRepository_GetByEmail_NotFound verifies "not found" for a missing email.
func TestRepository_GetByEmail_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := employee.NewRepository(pool)
	ctx := context.Background()

	_, err := repo.GetByEmail(ctx, "nobody@nowhere.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestRepository_CreateInvitation verifies code generation and expiry assignment.
func TestRepository_CreateInvitation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "emp-repo-inv")
	repo := employee.NewRepository(pool)
	ctx := context.Background()

	ttl := 48 * time.Hour
	inv, err := repo.CreateInvitation(ctx, uuid.MustParse(providerID), "New Employee", ttl)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, inv.ID)
	assert.NotEmpty(t, inv.Code)
	assert.Equal(t, "New Employee", inv.EmployeeName)
	assert.True(t, inv.ExpiresAt.After(time.Now()))
	assert.False(t, inv.CreatedAt.IsZero())
	assert.Nil(t, inv.UsedAt)
	assert.Nil(t, inv.UsedBy)
}

// TestRepository_CreateInvitation_UniqueCodes verifies that multiple invitations receive
// distinct codes.
func TestRepository_CreateInvitation_UniqueCodes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "emp-repo-inv-unique")
	repo := employee.NewRepository(pool)
	ctx := context.Background()

	ttl := 48 * time.Hour
	inv1, err := repo.CreateInvitation(ctx, uuid.MustParse(providerID), "Employee One", ttl)
	require.NoError(t, err)

	inv2, err := repo.CreateInvitation(ctx, uuid.MustParse(providerID), "Employee Two", ttl)
	require.NoError(t, err)

	assert.NotEqual(t, inv1.Code, inv2.Code)
}

// TestRepository_GetInvitationByCode verifies retrieval of a valid, unused invitation.
func TestRepository_GetInvitationByCode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "emp-repo-inv-bycode")
	repo := employee.NewRepository(pool)
	ctx := context.Background()

	inv, err := repo.CreateInvitation(ctx, uuid.MustParse(providerID), "Invitee", 48*time.Hour)
	require.NoError(t, err)

	got, err := repo.GetInvitationByCode(ctx, inv.Code)
	require.NoError(t, err)

	assert.Equal(t, inv.ID, got.ID)
	assert.Equal(t, inv.Code, got.Code)
	assert.Equal(t, "Invitee", got.EmployeeName)
}

// TestRepository_GetInvitationByCode_UsedInvitation verifies that a used invitation
// cannot be retrieved.
func TestRepository_GetInvitationByCode_UsedInvitation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "emp-repo-inv-used")
	empID := testutil.SeedEmployee(t, pool, providerID, "Employee")
	repo := employee.NewRepository(pool)
	ctx := context.Background()

	inv, err := repo.CreateInvitation(ctx, uuid.MustParse(providerID), "Invitee", 48*time.Hour)
	require.NoError(t, err)

	err = repo.MarkInvitationUsed(ctx, inv.ID, uuid.MustParse(empID))
	require.NoError(t, err)

	_, err = repo.GetInvitationByCode(ctx, inv.Code)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already used")
}

// TestRepository_GetInvitationByCode_Expired verifies that an expired invitation returns an error.
func TestRepository_GetInvitationByCode_Expired(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "emp-repo-inv-expired")
	repo := employee.NewRepository(pool)
	ctx := context.Background()

	// Insert invitation with a past expires_at directly via SQL
	pastExpiry := time.Now().Add(-1 * time.Hour)
	var invID uuid.UUID
	var code string
	err := pool.QueryRow(ctx, `
		INSERT INTO invitation_codes (provider_id, code, employee_name, expires_at)
		VALUES ($1, 'expiredcode', 'Old Invitee', $2)
		RETURNING id, code`,
		providerID, pastExpiry,
	).Scan(&invID, &code)
	require.NoError(t, err)

	_, err = repo.GetInvitationByCode(ctx, code)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

// TestRepository_MarkInvitationUsed verifies that used_at and used_by are set after marking.
func TestRepository_MarkInvitationUsed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "emp-repo-inv-markused")
	empID := testutil.SeedEmployee(t, pool, providerID, "Mark User")
	repo := employee.NewRepository(pool)
	ctx := context.Background()

	inv, err := repo.CreateInvitation(ctx, uuid.MustParse(providerID), "Invitee", 48*time.Hour)
	require.NoError(t, err)

	empUUID := uuid.MustParse(empID)
	err = repo.MarkInvitationUsed(ctx, inv.ID, empUUID)
	require.NoError(t, err)

	// Verify from DB that used_at and used_by are set
	var usedAt *time.Time
	var usedBy *uuid.UUID
	err = pool.QueryRow(ctx,
		`SELECT used_at, used_by FROM invitation_codes WHERE id = $1`,
		inv.ID,
	).Scan(&usedAt, &usedBy)
	require.NoError(t, err)

	require.NotNil(t, usedAt)
	require.NotNil(t, usedBy)
	assert.Equal(t, empUUID, *usedBy)
}

// TestRepository_ListInvitations verifies that invitations are returned ordered by created_at DESC.
func TestRepository_ListInvitations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "emp-repo-inv-list")
	repo := employee.NewRepository(pool)
	ctx := context.Background()

	ttl := 48 * time.Hour
	inv1, err := repo.CreateInvitation(ctx, uuid.MustParse(providerID), "First", ttl)
	require.NoError(t, err)

	// Small sleep to guarantee distinct created_at timestamps
	time.Sleep(5 * time.Millisecond)

	inv2, err := repo.CreateInvitation(ctx, uuid.MustParse(providerID), "Second", ttl)
	require.NoError(t, err)

	invitations, err := repo.ListInvitations(ctx, uuid.MustParse(providerID))
	require.NoError(t, err)

	require.Len(t, invitations, 2)
	// Most recent first
	assert.Equal(t, inv2.ID, invitations[0].ID)
	assert.Equal(t, inv1.ID, invitations[1].ID)
}

// TestRepository_ListInvitations_Empty verifies an empty list is returned for a provider
// with no invitations.
func TestRepository_ListInvitations_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "emp-repo-inv-list-empty")
	repo := employee.NewRepository(pool)
	ctx := context.Background()

	invitations, err := repo.ListInvitations(ctx, uuid.MustParse(providerID))
	require.NoError(t, err)
	assert.Empty(t, invitations)
}

// TestRepository_AssignServices verifies that services are assigned to an employee and that
// reassigning replaces the previous set.
func TestRepository_AssignServices(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "emp-repo-assign-svc")
	empID := testutil.SeedEmployee(t, pool, providerID, "Service Employee")
	svc1ID := testutil.SeedService(t, pool, providerID, "Haircut", 30)
	svc2ID := testutil.SeedService(t, pool, providerID, "Shave", 15)
	svc3ID := testutil.SeedService(t, pool, providerID, "Trim", 20)
	repo := employee.NewRepository(pool)
	ctx := context.Background()

	empUUID := uuid.MustParse(empID)
	svc1UUID := uuid.MustParse(svc1ID)
	svc2UUID := uuid.MustParse(svc2ID)
	svc3UUID := uuid.MustParse(svc3ID)

	// Assign first two services
	err := repo.AssignServices(ctx, empUUID, []uuid.UUID{svc1UUID, svc2UUID})
	require.NoError(t, err)

	ids, err := repo.ListServiceIDs(ctx, empUUID)
	require.NoError(t, err)
	assert.Len(t, ids, 2)
	assert.Contains(t, ids, svc1UUID)
	assert.Contains(t, ids, svc2UUID)

	// Reassign: only svc3 — svc1 and svc2 should be gone
	err = repo.AssignServices(ctx, empUUID, []uuid.UUID{svc3UUID})
	require.NoError(t, err)

	ids, err = repo.ListServiceIDs(ctx, empUUID)
	require.NoError(t, err)
	assert.Len(t, ids, 1)
	assert.Contains(t, ids, svc3UUID)
	assert.NotContains(t, ids, svc1UUID)
	assert.NotContains(t, ids, svc2UUID)
}

// TestRepository_ListServiceIDs_Empty verifies an empty slice when no services are assigned.
func TestRepository_ListServiceIDs_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	providerID := testutil.SeedProvider(t, pool, "emp-repo-svcids-empty")
	empID := testutil.SeedEmployee(t, pool, providerID, "No Services")
	repo := employee.NewRepository(pool)
	ctx := context.Background()

	ids, err := repo.ListServiceIDs(ctx, uuid.MustParse(empID))
	require.NoError(t, err)
	assert.Empty(t, ids)
}
