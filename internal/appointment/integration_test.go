package appointment_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/THEcanon001/turnero/internal/appointment"
	"github.com/THEcanon001/turnero/internal/schedule"
	"github.com/THEcanon001/turnero/internal/testutil"
)

// ---------- Repository tests ----------

func TestRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "apt-create")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp1")
	serviceID := testutil.SeedService(t, pool, providerID, "Corte", 30)

	repo := appointment.NewRepository(pool)

	pid, _ := uuid.Parse(providerID)
	eid, _ := uuid.Parse(employeeID)
	sid, _ := uuid.Parse(serviceID)

	apt := &appointment.Appointment{
		ProviderID:  pid,
		EmployeeID:  eid,
		ServiceID:   &sid,
		ClientName:  "Carlos Test",
		ClientPhone: "+5491100000000",
		Date:        "2026-10-15",
		StartTime:   "09:00",
		EndTime:     "09:30",
	}

	err := repo.Create(ctx, apt)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, apt.ID)
	assert.Equal(t, appointment.StatusConfirmed, apt.Status)
	assert.False(t, apt.CreatedAt.IsZero())
}

func TestRepository_Create_DuplicateSlot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "apt-dup")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
	serviceID := testutil.SeedService(t, pool, providerID, "Corte", 30)

	repo := appointment.NewRepository(pool)
	pid, _ := uuid.Parse(providerID)
	eid, _ := uuid.Parse(employeeID)
	sid, _ := uuid.Parse(serviceID)

	apt := &appointment.Appointment{
		ProviderID: pid, EmployeeID: eid, ServiceID: &sid,
		ClientName: "A", ClientPhone: "+1", Date: "2026-10-15",
		StartTime: "10:00", EndTime: "10:30",
	}
	require.NoError(t, repo.Create(ctx, apt))

	// Same slot, different client
	apt2 := &appointment.Appointment{
		ProviderID: pid, EmployeeID: eid, ServiceID: &sid,
		ClientName: "B", ClientPhone: "+2", Date: "2026-10-15",
		StartTime: "10:00", EndTime: "10:30",
	}
	err := repo.Create(ctx, apt2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "slot already taken")
}

func TestRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "apt-get")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
	serviceID := testutil.SeedService(t, pool, providerID, "Corte", 30)
	aptID := testutil.SeedAppointment(t, pool, providerID, employeeID, serviceID,
		"2026-10-15", "11:00", "11:30", "Client", "+123")

	repo := appointment.NewRepository(pool)
	id, _ := uuid.Parse(aptID)

	apt, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "Client", apt.ClientName)
	assert.Equal(t, appointment.StatusConfirmed, apt.Status)
	assert.Equal(t, "2026-10-15", apt.Date)
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := appointment.NewRepository(pool)
	_, err := repo.GetByID(context.Background(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "apt-list")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
	serviceID := testutil.SeedService(t, pool, providerID, "Corte", 30)

	testutil.SeedAppointment(t, pool, providerID, employeeID, serviceID,
		"2026-10-15", "09:00", "09:30", "A", "+1")
	testutil.SeedAppointment(t, pool, providerID, employeeID, serviceID,
		"2026-10-15", "10:00", "10:30", "B", "+2")
	testutil.SeedAppointment(t, pool, providerID, employeeID, serviceID,
		"2026-10-16", "09:00", "09:30", "C", "+3")

	repo := appointment.NewRepository(pool)
	pid, _ := uuid.Parse(providerID)

	// Filter by provider
	apts, total, err := repo.List(ctx, appointment.ListFilter{
		ProviderID: &pid,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, apts, 3)

	// Filter by date
	apts, total, err = repo.List(ctx, appointment.ListFilter{
		ProviderID: &pid,
		Date:       "2026-10-15",
	})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, apts, 2)

	// Filter by status
	status := appointment.StatusConfirmed
	apts, _, err = repo.List(ctx, appointment.ListFilter{
		ProviderID: &pid,
		Status:     &status,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, len(apts))
}

func TestRepository_List_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "apt-page")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
	serviceID := testutil.SeedService(t, pool, providerID, "Corte", 30)

	for i := 0; i < 5; i++ {
		start := fmt.Sprintf("%02d:00", 9+i)
		end := fmt.Sprintf("%02d:30", 9+i)
		testutil.SeedAppointment(t, pool, providerID, employeeID, serviceID,
			"2026-10-15", start, end, fmt.Sprintf("C%d", i), "+1")
	}

	repo := appointment.NewRepository(pool)
	pid, _ := uuid.Parse(providerID)

	apts, total, err := repo.List(ctx, appointment.ListFilter{
		ProviderID: &pid,
		Page:       1,
		PerPage:    2,
	})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, apts, 2)

	apts, _, err = repo.List(ctx, appointment.ListFilter{
		ProviderID: &pid,
		Page:       3,
		PerPage:    2,
	})
	require.NoError(t, err)
	assert.Len(t, apts, 1) // last page
}

func TestRepository_UpdateStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "apt-upd")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
	serviceID := testutil.SeedService(t, pool, providerID, "Corte", 30)
	aptID := testutil.SeedAppointment(t, pool, providerID, employeeID, serviceID,
		"2026-10-15", "09:00", "09:30", "Client", "+1")

	repo := appointment.NewRepository(pool)
	id, _ := uuid.Parse(aptID)

	err := repo.UpdateStatus(ctx, id, appointment.StatusCompleted, nil)
	require.NoError(t, err)

	apt, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, appointment.StatusCompleted, apt.Status)
}

func TestRepository_UpdateStatus_WithReason(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "apt-reas")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
	serviceID := testutil.SeedService(t, pool, providerID, "Corte", 30)
	aptID := testutil.SeedAppointment(t, pool, providerID, employeeID, serviceID,
		"2026-10-15", "09:00", "09:30", "Client", "+1")

	repo := appointment.NewRepository(pool)
	id, _ := uuid.Parse(aptID)

	reason := "client requested"
	err := repo.UpdateStatus(ctx, id, appointment.StatusCancelled, &reason)
	require.NoError(t, err)

	apt, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, appointment.StatusCancelled, apt.Status)
	require.NotNil(t, apt.CancellationReason)
	assert.Equal(t, "client requested", *apt.CancellationReason)
}

func TestRepository_UpdateStatus_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := appointment.NewRepository(pool)
	err := repo.UpdateStatus(context.Background(), uuid.New(), appointment.StatusCompleted, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRepository_UpdateEmployee(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "apt-emp")
	emp1ID := testutil.SeedEmployee(t, pool, providerID, "Emp1")
	emp2ID := testutil.SeedEmployee(t, pool, providerID, "Emp2")
	serviceID := testutil.SeedService(t, pool, providerID, "Corte", 30)
	aptID := testutil.SeedAppointment(t, pool, providerID, emp1ID, serviceID,
		"2026-10-15", "09:00", "09:30", "Client", "+1")

	repo := appointment.NewRepository(pool)
	id, _ := uuid.Parse(aptID)
	newEmpID, _ := uuid.Parse(emp2ID)

	err := repo.UpdateEmployee(ctx, id, newEmpID)
	require.NoError(t, err)

	apt, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, newEmpID, apt.EmployeeID)
}

func TestRepository_CancelByEmployeeAndDateRange(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "apt-batch")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
	serviceID := testutil.SeedService(t, pool, providerID, "Corte", 30)

	testutil.SeedAppointment(t, pool, providerID, employeeID, serviceID,
		"2026-10-15", "09:00", "09:30", "A", "+1")
	testutil.SeedAppointment(t, pool, providerID, employeeID, serviceID,
		"2026-10-15", "10:00", "10:30", "B", "+2")
	testutil.SeedAppointment(t, pool, providerID, employeeID, serviceID,
		"2026-10-16", "09:00", "09:30", "C", "+3")

	repo := appointment.NewRepository(pool)
	eid, _ := uuid.Parse(employeeID)

	n, err := repo.CancelByEmployeeAndDateRange(ctx, eid, "2026-10-15", "2026-10-15", "day off")
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)

	// Verify: the 10/16 appointment is untouched
	pid, _ := uuid.Parse(providerID)
	apts, _, err := repo.List(ctx, appointment.ListFilter{ProviderID: &pid, Date: "2026-10-16"})
	require.NoError(t, err)
	require.Len(t, apts, 1)
	assert.Equal(t, appointment.StatusConfirmed, apts[0].Status)
}

func TestRepository_GetStats(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "apt-stats")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
	serviceID := testutil.SeedService(t, pool, providerID, "Corte", 30)

	apt1 := testutil.SeedAppointment(t, pool, providerID, employeeID, serviceID,
		"2026-10-15", "09:00", "09:30", "A", "+1")
	testutil.SeedAppointment(t, pool, providerID, employeeID, serviceID,
		"2026-10-15", "10:00", "10:30", "B", "+2")

	repo := appointment.NewRepository(pool)
	id1, _ := uuid.Parse(apt1)
	require.NoError(t, repo.UpdateStatus(ctx, id1, appointment.StatusCompleted, nil))

	pid, _ := uuid.Parse(providerID)
	stats, err := repo.GetStats(ctx, pid, "2026-10-01", "2026-10-31")
	require.NoError(t, err)
	assert.Equal(t, 2, stats.Total)
	assert.Equal(t, 1, stats.Completed)
	assert.Equal(t, 1, stats.Confirmed)
}

func TestRepository_GetStatsByEmployee(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "apt-empstats")
	emp1ID := testutil.SeedEmployee(t, pool, providerID, "Emp1")
	emp2ID := testutil.SeedEmployee(t, pool, providerID, "Emp2")
	serviceID := testutil.SeedService(t, pool, providerID, "Corte", 30)

	testutil.SeedAppointment(t, pool, providerID, emp1ID, serviceID,
		"2026-10-15", "09:00", "09:30", "A", "+1")
	testutil.SeedAppointment(t, pool, providerID, emp1ID, serviceID,
		"2026-10-15", "10:00", "10:30", "B", "+2")
	testutil.SeedAppointment(t, pool, providerID, emp2ID, serviceID,
		"2026-10-15", "11:00", "11:30", "C", "+3")

	repo := appointment.NewRepository(pool)
	pid, _ := uuid.Parse(providerID)

	empStats, err := repo.GetStatsByEmployee(ctx, pid, "2026-10-01", "2026-10-31")
	require.NoError(t, err)
	assert.Len(t, empStats, 2)
}

func TestRepository_ListByEmployeeAndDate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "apt-empdate")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
	serviceID := testutil.SeedService(t, pool, providerID, "Corte", 30)

	testutil.SeedAppointment(t, pool, providerID, employeeID, serviceID,
		"2026-10-15", "09:00", "09:30", "A", "+1")
	testutil.SeedAppointment(t, pool, providerID, employeeID, serviceID,
		"2026-10-15", "10:00", "10:30", "B", "+2")

	repo := appointment.NewRepository(pool)
	eid, _ := uuid.Parse(employeeID)

	apts, err := repo.ListByEmployeeAndDate(ctx, eid, "2026-10-15")
	require.NoError(t, err)
	assert.Len(t, apts, 2)
	assert.Equal(t, "09:00:00", apts[0].StartTime) // ordered ASC
}

// ---------- Service tests ----------

func TestService_GetAvailableSlots(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "apt-slots")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")

	// Wednesday 2026-10-14 => day_of_week=3
	testutil.SeedSchedule(t, pool, employeeID, 3, "09:00", "11:00", 30)

	scheduleRepo := schedule.NewRepository(pool)
	aptRepo := appointment.NewRepository(pool)
	svc := appointment.NewService(aptRepo, scheduleRepo)

	slots, err := svc.GetAvailableSlots(ctx, mustParseUUID(t, employeeID), "2026-10-14")
	require.NoError(t, err)
	assert.Len(t, slots, 4) // 09:00, 09:30, 10:00, 10:30
	for _, s := range slots {
		assert.True(t, s.Available)
	}
}

func TestService_GetAvailableSlots_WithBookedSlot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "apt-booked")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
	serviceID := testutil.SeedService(t, pool, providerID, "Corte", 30)

	// Wednesday 2026-10-14
	testutil.SeedSchedule(t, pool, employeeID, 3, "09:00", "11:00", 30)
	testutil.SeedAppointment(t, pool, providerID, employeeID, serviceID,
		"2026-10-14", "09:30", "10:00", "Booked", "+1")

	scheduleRepo := schedule.NewRepository(pool)
	aptRepo := appointment.NewRepository(pool)
	svc := appointment.NewService(aptRepo, scheduleRepo)

	slots, err := svc.GetAvailableSlots(ctx, mustParseUUID(t, employeeID), "2026-10-14")
	require.NoError(t, err)

	for _, s := range slots {
		if s.StartTime == "09:30" {
			assert.False(t, s.Available, "booked slot should not be available")
		} else {
			assert.True(t, s.Available)
		}
	}
}

func TestService_GetAvailableSlots_NoSchedule(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "apt-nosched")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")

	scheduleRepo := schedule.NewRepository(pool)
	aptRepo := appointment.NewRepository(pool)
	svc := appointment.NewService(aptRepo, scheduleRepo)

	slots, err := svc.GetAvailableSlots(ctx, mustParseUUID(t, employeeID), "2026-10-14")
	require.NoError(t, err)
	assert.Empty(t, slots)
}

func TestService_GetAvailableSlots_DayOff(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "apt-dayoff")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
	testutil.SeedSchedule(t, pool, employeeID, 3, "09:00", "18:00", 30) // Wednesday

	// Add day-off exception
	scheduleRepo := schedule.NewRepository(pool)
	eid := mustParseUUID(t, employeeID)
	exc := &schedule.ScheduleException{
		EmployeeID:  eid,
		Date:        "2026-10-14",
		IsAvailable: false,
	}
	require.NoError(t, scheduleRepo.CreateException(ctx, exc))

	aptRepo := appointment.NewRepository(pool)
	svc := appointment.NewService(aptRepo, scheduleRepo)

	slots, err := svc.GetAvailableSlots(ctx, eid, "2026-10-14")
	require.NoError(t, err)
	assert.Empty(t, slots)
}

func TestService_GetAvailableSlots_WithBreak(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "apt-break")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")

	// 30 min slot + 10 min break = 40 min per slot
	// 09:00-10:20 => slots: 09:00-09:30, 09:40-10:10 (2 slots)
	eid := mustParseUUID(t, employeeID)
	scheduleRepo := schedule.NewRepository(pool)
	s := &schedule.Schedule{
		EmployeeID:            eid,
		DayOfWeek:             3,
		StartTime:             "09:00",
		EndTime:               "10:20",
		SlotDurationMinutes:   30,
		BreakAfterSlotMinutes: 10,
	}
	require.NoError(t, scheduleRepo.Upsert(ctx, s))

	aptRepo := appointment.NewRepository(pool)
	svc := appointment.NewService(aptRepo, scheduleRepo)

	slots, err := svc.GetAvailableSlots(ctx, eid, "2026-10-14") // Wednesday
	require.NoError(t, err)
	assert.Len(t, slots, 2)
	assert.Equal(t, "09:00", slots[0].StartTime)
	assert.Equal(t, "09:30", slots[0].EndTime)
	assert.Equal(t, "09:40", slots[1].StartTime)
	assert.Equal(t, "10:10", slots[1].EndTime)
}

func TestService_GetAvailableSlots_InvalidDate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	scheduleRepo := schedule.NewRepository(pool)
	aptRepo := appointment.NewRepository(pool)
	svc := appointment.NewService(aptRepo, scheduleRepo)

	_, err := svc.GetAvailableSlots(context.Background(), uuid.New(), "not-a-date")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid date")
}

func mustParseUUID(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	require.NoError(t, err)
	return id
}
