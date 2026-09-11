package schedule_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/THEcanon001/turnero/internal/schedule"
	"github.com/THEcanon001/turnero/internal/testutil"
)

// ---------- Repository tests ----------

func TestRepository_Upsert_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "sched-upsert")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
	eid := mustParseUUID(t, employeeID)

	repo := schedule.NewRepository(pool)

	s := &schedule.Schedule{
		EmployeeID:            eid,
		DayOfWeek:             1, // Monday
		StartTime:             "09:00",
		EndTime:               "18:00",
		SlotDurationMinutes:   30,
		BreakAfterSlotMinutes: 0,
	}

	err := repo.Upsert(ctx, s)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, s.ID)
	assert.True(t, s.IsActive)
	assert.False(t, s.CreatedAt.IsZero())
}

func TestRepository_Upsert_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "sched-upd")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
	eid := mustParseUUID(t, employeeID)

	repo := schedule.NewRepository(pool)

	s := &schedule.Schedule{
		EmployeeID: eid, DayOfWeek: 1,
		StartTime: "09:00", EndTime: "17:00",
		SlotDurationMinutes: 30,
	}
	require.NoError(t, repo.Upsert(ctx, s))
	originalID := s.ID

	// Upsert same day => update
	s2 := &schedule.Schedule{
		EmployeeID: eid, DayOfWeek: 1,
		StartTime: "10:00", EndTime: "18:00",
		SlotDurationMinutes: 45,
	}
	require.NoError(t, repo.Upsert(ctx, s2))
	assert.Equal(t, originalID, s2.ID) // same row

	// Verify updated values
	schedules, err := repo.ListByEmployee(ctx, eid)
	require.NoError(t, err)
	require.Len(t, schedules, 1)
	assert.Equal(t, "10:00:00", schedules[0].StartTime)
	assert.Equal(t, int16(45), schedules[0].SlotDurationMinutes)
}

func TestRepository_ListByEmployee(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "sched-list")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
	eid := mustParseUUID(t, employeeID)

	repo := schedule.NewRepository(pool)

	// Insert Mon-Fri
	for day := int16(1); day <= 5; day++ {
		s := &schedule.Schedule{
			EmployeeID: eid, DayOfWeek: day,
			StartTime: "09:00", EndTime: "18:00",
			SlotDurationMinutes: 30,
		}
		require.NoError(t, repo.Upsert(ctx, s))
	}

	schedules, err := repo.ListByEmployee(ctx, eid)
	require.NoError(t, err)
	assert.Len(t, schedules, 5)
	// Ordered by day_of_week ASC
	assert.Equal(t, int16(1), schedules[0].DayOfWeek)
	assert.Equal(t, int16(5), schedules[4].DayOfWeek)
}

func TestRepository_ListByEmployee_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := schedule.NewRepository(pool)
	schedules, err := repo.ListByEmployee(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Nil(t, schedules)
}

func TestRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "sched-del")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
	eid := mustParseUUID(t, employeeID)

	repo := schedule.NewRepository(pool)
	s := &schedule.Schedule{
		EmployeeID: eid, DayOfWeek: 1,
		StartTime: "09:00", EndTime: "18:00",
		SlotDurationMinutes: 30,
	}
	require.NoError(t, repo.Upsert(ctx, s))

	err := repo.Delete(ctx, s.ID)
	require.NoError(t, err)

	schedules, err := repo.ListByEmployee(ctx, eid)
	require.NoError(t, err)
	assert.Empty(t, schedules)
}

func TestRepository_Delete_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := schedule.NewRepository(pool)
	err := repo.Delete(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------- Exception tests ----------

func TestRepository_CreateException(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "sched-exc")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
	eid := mustParseUUID(t, employeeID)

	repo := schedule.NewRepository(pool)

	reason := "vacation"
	exc := &schedule.ScheduleException{
		EmployeeID:  eid,
		Date:        "2026-12-25",
		IsAvailable: false,
		Reason:      &reason,
	}

	err := repo.CreateException(ctx, exc)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, exc.ID)
	assert.False(t, exc.CreatedAt.IsZero())
}

func TestRepository_CreateException_Upsert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "sched-exc2")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
	eid := mustParseUUID(t, employeeID)

	repo := schedule.NewRepository(pool)

	reason1 := "holiday"
	exc1 := &schedule.ScheduleException{
		EmployeeID: eid, Date: "2026-12-25", IsAvailable: false, Reason: &reason1,
	}
	require.NoError(t, repo.CreateException(ctx, exc1))

	// Upsert same date
	reason2 := "special hours"
	startTime := "10:00"
	endTime := "14:00"
	exc2 := &schedule.ScheduleException{
		EmployeeID: eid, Date: "2026-12-25", IsAvailable: true,
		StartTime: &startTime, EndTime: &endTime, Reason: &reason2,
	}
	require.NoError(t, repo.CreateException(ctx, exc2))
	assert.Equal(t, exc1.ID, exc2.ID) // same row

	// Verify
	found, err := repo.GetExceptionByID(ctx, exc2.ID)
	require.NoError(t, err)
	assert.True(t, found.IsAvailable)
	require.NotNil(t, found.Reason)
	assert.Equal(t, "special hours", *found.Reason)
}

func TestRepository_ListExceptions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "sched-lexc")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
	eid := mustParseUUID(t, employeeID)

	repo := schedule.NewRepository(pool)

	for _, date := range []string{"2026-12-24", "2026-12-25", "2026-12-31"} {
		exc := &schedule.ScheduleException{
			EmployeeID: eid, Date: date, IsAvailable: false,
		}
		require.NoError(t, repo.CreateException(ctx, exc))
	}

	// Query subset
	exceptions, err := repo.ListExceptions(ctx, eid, "2026-12-24", "2026-12-25")
	require.NoError(t, err)
	assert.Len(t, exceptions, 2)

	// All
	exceptions, err = repo.ListExceptions(ctx, eid, "2026-12-01", "2026-12-31")
	require.NoError(t, err)
	assert.Len(t, exceptions, 3)
}

func TestRepository_GetExceptionByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "sched-getexc")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
	eid := mustParseUUID(t, employeeID)

	repo := schedule.NewRepository(pool)
	exc := &schedule.ScheduleException{
		EmployeeID: eid, Date: "2026-12-25", IsAvailable: false,
	}
	require.NoError(t, repo.CreateException(ctx, exc))

	found, err := repo.GetExceptionByID(ctx, exc.ID)
	require.NoError(t, err)
	assert.Equal(t, eid, found.EmployeeID)
	assert.Equal(t, "2026-12-25", found.Date)
}

func TestRepository_GetExceptionByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := schedule.NewRepository(pool)
	_, err := repo.GetExceptionByID(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRepository_DeleteException(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "sched-delexc")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
	eid := mustParseUUID(t, employeeID)

	repo := schedule.NewRepository(pool)
	exc := &schedule.ScheduleException{
		EmployeeID: eid, Date: "2026-12-25", IsAvailable: false,
	}
	require.NoError(t, repo.CreateException(ctx, exc))

	err := repo.DeleteException(ctx, exc.ID)
	require.NoError(t, err)

	_, err = repo.GetExceptionByID(ctx, exc.ID)
	require.Error(t, err)
}

func TestRepository_DeleteException_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := schedule.NewRepository(pool)
	err := repo.DeleteException(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------- validateTimeRange unit tests ----------

func TestValidateTimeRange(t *testing.T) {
	tests := []struct {
		name      string
		start     string
		end       string
		wantError string
	}{
		{"valid range", "09:00", "18:00", ""},
		{"same time", "09:00", "09:00", "start_time must be before end_time"},
		{"end before start", "18:00", "09:00", "start_time must be before end_time"},
		{"invalid start", "25:00", "18:00", "invalid start_time"},
		{"invalid end", "09:00", "abc", "invalid end_time"},
		{"empty start", "", "18:00", "invalid start_time"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := schedule.ValidateTimeRange(tt.start, tt.end)
			if tt.wantError == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantError)
			}
		})
	}
}

func mustParseUUID(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	require.NoError(t, err)
	return id
}
