package billing_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/THEcanon001/turnero/internal/billing"
	"github.com/THEcanon001/turnero/internal/testutil"
)

// mockNotifSender records all notifications sent via SendToProvider.
type mockNotifSender struct {
	mu   sync.Mutex
	sent []sentMsg
}

type sentMsg struct {
	providerID uuid.UUID
	msg        billing.NotificationMessage
}

func (m *mockNotifSender) SendToProvider(ctx context.Context, providerID uuid.UUID, msg billing.NotificationMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, sentMsg{providerID, msg})
	return nil
}

func (m *mockNotifSender) messages() []sentMsg {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]sentMsg, len(m.sent))
	copy(out, m.sent)
	return out
}

func TestWorker_RunMonthlyCycle_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)
	svc := billing.NewService(repo)
	worker := billing.NewWorker(pool, svc, repo, nil)

	// Seed 2 active providers with appointments marked as 'completed'
	providerID1 := testutil.SeedProvider(t, pool, "worker-cycle-p1")
	providerID2 := testutil.SeedProvider(t, pool, "worker-cycle-p2")

	providerUUID1 := uuid.MustParse(providerID1)
	providerUUID2 := uuid.MustParse(providerID2)

	// Seed appointments for provider 1
	emp1 := testutil.SeedEmployee(t, pool, providerID1, "Emp1")
	svc1 := testutil.SeedService(t, pool, providerID1, "Svc1", 30)
	testutil.SeedAppointment(t, pool, providerID1, emp1, svc1, "2026-09-11", "09:00", "09:30", "ClientA", "+1001")
	testutil.SeedAppointment(t, pool, providerID1, emp1, svc1, "2026-09-11", "10:00", "10:30", "ClientB", "+1002")

	_, err := pool.Exec(ctx, "UPDATE appointments SET status = 'completed' WHERE provider_id = $1", providerUUID1)
	require.NoError(t, err)

	// Seed appointments for provider 2
	emp2 := testutil.SeedEmployee(t, pool, providerID2, "Emp2")
	svc2 := testutil.SeedService(t, pool, providerID2, "Svc2", 45)
	testutil.SeedAppointment(t, pool, providerID2, emp2, svc2, "2026-09-11", "14:00", "14:45", "ClientC", "+2001")

	_, err = pool.Exec(ctx, "UPDATE appointments SET status = 'completed' WHERE provider_id = $1", providerUUID2)
	require.NoError(t, err)

	// Run the monthly cycle
	err = worker.RunMonthlyCycle(ctx)
	require.NoError(t, err)

	// Verify billing_usage rows were created and synced for provider 1
	now := time.Now()
	usage1, err := repo.GetUsage(ctx, providerUUID1, now)
	require.NoError(t, err)
	assert.Equal(t, 2, usage1.CompletedAppointments)
	assert.Equal(t, 0, usage1.BillableAppointments) // 2 < 20 free tier
	assert.Equal(t, 0.0, usage1.AmountDue)

	// Verify billing_usage rows were created and synced for provider 2
	usage2, err := repo.GetUsage(ctx, providerUUID2, now)
	require.NoError(t, err)
	assert.Equal(t, 1, usage2.CompletedAppointments)
	assert.Equal(t, 0, usage2.BillableAppointments)
	assert.Equal(t, 0.0, usage2.AmountDue)
}

func TestWorker_RunMonthlyCycle_BeyondFreeTier_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)
	svc := billing.NewService(repo)
	worker := billing.NewWorker(pool, svc, repo, nil)

	providerID := testutil.SeedProvider(t, pool, "worker-cycle-beyond")
	providerUUID := uuid.MustParse(providerID)
	emp := testutil.SeedEmployee(t, pool, providerID, "EmpBeyond")
	service := testutil.SeedService(t, pool, providerID, "SvcBeyond", 15)

	// Seed 25 appointments in September 2026
	for i := 0; i < 25; i++ {
		hour := 8 + (i / 4)
		minute := (i % 4) * 15
		startTime := time.Date(2026, 9, 11, hour, minute, 0, 0, time.UTC).Format("15:04")
		endTime := time.Date(2026, 9, 11, hour, minute+15, 0, 0, time.UTC).Format("15:04")
		testutil.SeedAppointment(t, pool, providerID, emp, service, "2026-09-11", startTime, endTime, "Client", "+999")
	}

	_, err := pool.Exec(ctx, "UPDATE appointments SET status = 'completed' WHERE provider_id = $1", providerUUID)
	require.NoError(t, err)

	err = worker.RunMonthlyCycle(ctx)
	require.NoError(t, err)

	now := time.Now()
	usage, err := repo.GetUsage(ctx, providerUUID, now)
	require.NoError(t, err)

	assert.Equal(t, 25, usage.CompletedAppointments)
	// 25 - 20 = 5 billable
	assert.Equal(t, 5, usage.BillableAppointments)
	// 5 * 0.50 = 2.50
	assert.InDelta(t, 2.50, usage.AmountDue, 0.001)
}

func TestWorker_RunMonthlyCycle_NoProviders_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)
	svc := billing.NewService(repo)
	worker := billing.NewWorker(pool, svc, repo, nil)

	// No providers seeded — cycle should succeed with no rows processed
	err := worker.RunMonthlyCycle(ctx)
	require.NoError(t, err)
}

func TestWorker_RunInvoiceNotifications_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)
	svc := billing.NewService(repo)
	notif := &mockNotifSender{}
	worker := billing.NewWorker(pool, svc, repo, notif)

	providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "worker-invoice-notif"))

	// Create a usage row with amount_due > 0 and is_paid = false
	usage, err := repo.GetOrCreateUsage(ctx, providerID, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.NoError(t, repo.SyncUsageTotals(ctx, usage.ID, 25)) // 2.50 due

	err = worker.RunInvoiceNotifications(ctx)
	require.NoError(t, err)

	msgs := notif.messages()
	require.Len(t, msgs, 1)

	msg := msgs[0]
	assert.Equal(t, providerID, msg.providerID)
	assert.Equal(t, "Factura pendiente", msg.msg.Title)
	assert.Contains(t, msg.msg.Body, "2.50")
	assert.Contains(t, msg.msg.Body, "MXN")
	assert.Contains(t, msg.msg.Body, "2026-09")
	assert.Equal(t, "billing_invoice", msg.msg.Data["type"])
	assert.Equal(t, "2026-09", msg.msg.Data["month"])
	assert.Equal(t, "2.50", msg.msg.Data["amount"])
	assert.Equal(t, "MXN", msg.msg.Data["currency"])
}

func TestWorker_RunInvoiceNotifications_MultipleProviders_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)
	svc := billing.NewService(repo)
	notif := &mockNotifSender{}
	worker := billing.NewWorker(pool, svc, repo, notif)

	providerID1 := uuid.MustParse(testutil.SeedProvider(t, pool, "worker-invoice-multi-1"))
	providerID2 := uuid.MustParse(testutil.SeedProvider(t, pool, "worker-invoice-multi-2"))
	providerID3 := uuid.MustParse(testutil.SeedProvider(t, pool, "worker-invoice-multi-3"))

	// Provider 1: unpaid with amount_due > 0
	u1, err := repo.GetOrCreateUsage(ctx, providerID1, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.NoError(t, repo.SyncUsageTotals(ctx, u1.ID, 25))

	// Provider 2: paid (should not get notification)
	u2, err := repo.GetOrCreateUsage(ctx, providerID2, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.NoError(t, repo.SyncUsageTotals(ctx, u2.ID, 25))
	require.NoError(t, repo.MarkPaid(ctx, u2.ID))

	// Provider 3: amount_due = 0 (under free tier, should not get notification)
	u3, err := repo.GetOrCreateUsage(ctx, providerID3, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.NoError(t, repo.SyncUsageTotals(ctx, u3.ID, 5))

	err = worker.RunInvoiceNotifications(ctx)
	require.NoError(t, err)

	msgs := notif.messages()
	// Only provider 1 should have received a notification
	require.Len(t, msgs, 1)
	assert.Equal(t, providerID1, msgs[0].providerID)
}

func TestWorker_RunInvoiceNotifications_NilNotifService_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)
	svc := billing.NewService(repo)

	// Worker with nil notifService
	worker := billing.NewWorker(pool, svc, repo, nil)

	// Seed a provider with unpaid billing — but since notifService is nil, should return nil immediately
	providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "worker-invoice-nil-notif"))
	usage, err := repo.GetOrCreateUsage(ctx, providerID, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.NoError(t, repo.SyncUsageTotals(ctx, usage.ID, 25))

	err = worker.RunInvoiceNotifications(ctx)
	assert.NoError(t, err)
}

func TestWorker_RunInvoiceNotifications_NoUnpaid_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)
	svc := billing.NewService(repo)
	notif := &mockNotifSender{}
	worker := billing.NewWorker(pool, svc, repo, notif)

	// No usage rows at all
	err := worker.RunInvoiceNotifications(ctx)
	require.NoError(t, err)

	msgs := notif.messages()
	assert.Empty(t, msgs)
}
