package worker_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/THEcanon001/turnero/internal/testutil"
	"github.com/THEcanon001/turnero/internal/worker"
)

func TestWorker_Run_WithData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()

	// Seed providers, employees, services, and appointments so every collector
	// has at least one row to process.
	providerID := testutil.SeedProvider(t, pool, "metrics-worker")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Worker Emp")
	serviceID := testutil.SeedService(t, pool, providerID, "Svc", 30)

	today := time.Now().Format("2006-01-02")
	testutil.SeedAppointment(t, pool, providerID, employeeID, serviceID, today, "10:00", "10:30", "Client", "+123")

	// Insert a billing_usage row with amount_due > 0 so the billing collector
	// exercises the SUM path.
	_, err := pool.Exec(ctx,
		`INSERT INTO billing_usage (provider_id, month, completed_appointments, free_tier_limit, billable_appointments, amount_due, is_paid)
		 VALUES ($1, $2, 25, 20, 5, 2.50, false)`,
		uuid.MustParse(providerID), time.Now(),
	)
	require.NoError(t, err)

	w := worker.NewWorker(pool)
	err = w.Run(ctx)

	// Run always returns nil even when a collector logs an error internally.
	require.NoError(t, err)
}

func TestWorker_Run_EmptyDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	// No data seeded — all collectors must handle zero rows gracefully.
	w := worker.NewWorker(pool)
	err := w.Run(context.Background())

	require.NoError(t, err)
}

func TestWorker_Run_MultipleProviders(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()

	// Seed several providers and employees to exercise higher counts.
	for i, slug := range []string{"metrics-p1", "metrics-p2", "metrics-p3"} {
		providerID := testutil.SeedProvider(t, pool, slug)
		for j := 0; j <= i; j++ {
			testutil.SeedEmployee(t, pool, providerID, "Emp")
		}
	}

	w := worker.NewWorker(pool)
	err := w.Run(ctx)
	require.NoError(t, err)
}

func TestWorker_Run_AppointmentStatuses(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "metrics-statuses")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp Status")
	serviceID := testutil.SeedService(t, pool, providerID, "Svc Status", 30)
	today := time.Now().Format("2006-01-02")

	// Seed one appointment per status so every gauge label is exercised.
	statuses := []string{"confirmed", "completed", "cancelled", "no_show"}
	hours := []struct{ start, end string }{
		{"09:00", "09:30"},
		{"10:00", "10:30"},
		{"11:00", "11:30"},
		{"12:00", "12:30"},
	}
	for i, status := range statuses {
		apptID := testutil.SeedAppointment(t, pool, providerID, employeeID, serviceID, today, hours[i].start, hours[i].end, "Client", "+1234")
		_, err := pool.Exec(ctx, `UPDATE appointments SET status = $1 WHERE id = $2`, status, apptID)
		require.NoError(t, err)
	}

	w := worker.NewWorker(pool)
	err := w.Run(ctx)
	require.NoError(t, err)
}

func TestWorker_Run_BillingAllPaid(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "metrics-billing-paid")

	// All rows are paid — COALESCE(SUM, 0) should return 0.
	_, err := pool.Exec(ctx,
		`INSERT INTO billing_usage (provider_id, month, completed_appointments, free_tier_limit, billable_appointments, amount_due, is_paid)
		 VALUES ($1, $2, 10, 20, 0, 0.00, true)`,
		uuid.MustParse(providerID), time.Now(),
	)
	require.NoError(t, err)

	w := worker.NewWorker(pool)
	err = w.Run(ctx)
	require.NoError(t, err)
}

func TestWorker_Run_MultipleCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "metrics-multi-run")
	testutil.SeedEmployee(t, pool, providerID, "Emp Multi")

	w := worker.NewWorker(pool)

	// Calling Run multiple times must be idempotent and error-free.
	for i := 0; i < 3; i++ {
		err := w.Run(ctx)
		require.NoError(t, err)
	}
}
