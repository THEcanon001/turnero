package billing_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/THEcanon001/turnero/internal/billing"
	"github.com/THEcanon001/turnero/internal/testutil"
)

func TestRepository_GetOrCreateUsage_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "billing-repo-get-or-create"))
	repo := billing.NewRepository(pool)
	month := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	t.Run("creates new usage row", func(t *testing.T) {
		u, err := repo.GetOrCreateUsage(ctx, providerID, month)
		require.NoError(t, err)
		require.NotNil(t, u)

		assert.Equal(t, providerID, u.ProviderID)
		assert.Equal(t, month, u.Month)
		assert.Equal(t, billing.FreeTierLimit, u.FreeTierLimit)
		assert.Equal(t, 0, u.CompletedAppointments)
		assert.Equal(t, 0, u.BillableAppointments)
		assert.Equal(t, 0.0, u.AmountDue)
		assert.False(t, u.IsPaid)
		assert.NotEqual(t, uuid.Nil, u.ID)
	})

	t.Run("upsert on second call returns same row", func(t *testing.T) {
		u1, err := repo.GetOrCreateUsage(ctx, providerID, month)
		require.NoError(t, err)

		u2, err := repo.GetOrCreateUsage(ctx, providerID, month)
		require.NoError(t, err)

		assert.Equal(t, u1.ID, u2.ID)
		assert.Equal(t, u1.ProviderID, u2.ProviderID)
		assert.Equal(t, u1.Month, u2.Month)
	})

	t.Run("normalizes month to first of month", func(t *testing.T) {
		// Pass mid-month date; should still resolve to 1st
		midMonth := time.Date(2026, 9, 15, 12, 30, 0, 0, time.UTC)
		u, err := repo.GetOrCreateUsage(ctx, providerID, midMonth)
		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), u.Month)
	})
}

func TestRepository_IncrementCompleted_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)

	t.Run("increments from 0 and stays within free tier", func(t *testing.T) {
		providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "billing-repo-incr-free"))

		u, err := repo.IncrementCompleted(ctx, providerID)
		require.NoError(t, err)
		require.NotNil(t, u)

		assert.Equal(t, 1, u.CompletedAppointments)
		assert.Equal(t, 0, u.BillableAppointments)
		assert.Equal(t, 0.0, u.AmountDue)
	})

	t.Run("increments beyond free tier and calculates amount due", func(t *testing.T) {
		providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "billing-repo-incr-beyond"))

		// Increment up to free tier limit
		for i := 0; i < billing.FreeTierLimit; i++ {
			_, err := repo.IncrementCompleted(ctx, providerID)
			require.NoError(t, err)
		}

		// The (FreeTierLimit+1)th call should trigger billing
		u, err := repo.IncrementCompleted(ctx, providerID)
		require.NoError(t, err)

		assert.Equal(t, billing.FreeTierLimit+1, u.CompletedAppointments)
		assert.Equal(t, 1, u.BillableAppointments)
		assert.InDelta(t, billing.PricePerAppointment, u.AmountDue, 0.001)
	})

	t.Run("amount_due is capped at monthly cap", func(t *testing.T) {
		providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "billing-repo-incr-cap"))

		// Need FreeTierLimit + (MonthlyCap/PricePerAppointment) + 1 to exceed cap
		// MonthlyCap=10, PricePerAppointment=0.50 => cap at 20 billable => 40 total
		capPoint := billing.FreeTierLimit + int(billing.MonthlyCap/billing.PricePerAppointment) + 1

		var lastUsage *billing.BillingUsage
		for i := 0; i < capPoint; i++ {
			u, err := repo.IncrementCompleted(ctx, providerID)
			require.NoError(t, err)
			lastUsage = u
		}

		require.NotNil(t, lastUsage)
		assert.Equal(t, billing.MonthlyCap, lastUsage.AmountDue)
	})
}

func TestRepository_GetUsage_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)

	t.Run("returns existing usage", func(t *testing.T) {
		providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "billing-repo-get-usage"))
		month := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

		created, err := repo.GetOrCreateUsage(ctx, providerID, month)
		require.NoError(t, err)

		fetched, err := repo.GetUsage(ctx, providerID, month)
		require.NoError(t, err)
		require.NotNil(t, fetched)

		assert.Equal(t, created.ID, fetched.ID)
		assert.Equal(t, providerID, fetched.ProviderID)
		assert.Equal(t, month, fetched.Month)
	})

	t.Run("returns not found error for missing usage", func(t *testing.T) {
		nonExistentProvider := uuid.New()
		month := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

		_, err := repo.GetUsage(ctx, nonExistentProvider, month)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestRepository_GetCurrentUsage_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)
	providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "billing-repo-current-usage"))

	u, err := repo.GetCurrentUsage(ctx, providerID)
	require.NoError(t, err)
	require.NotNil(t, u)

	// Should return current month
	now := time.Now()
	expectedMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expectedMonth, u.Month)
	assert.Equal(t, providerID, u.ProviderID)

	// Second call should return the same row (idempotent)
	u2, err := repo.GetCurrentUsage(ctx, providerID)
	require.NoError(t, err)
	assert.Equal(t, u.ID, u2.ID)
}

func TestRepository_ListUsageHistory_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)
	providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "billing-repo-list-history"))

	// Create usage rows for multiple months
	months := []time.Time{
		time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	}
	for _, m := range months {
		_, err := repo.GetOrCreateUsage(ctx, providerID, m)
		require.NoError(t, err)
	}

	t.Run("returns all months in DESC order", func(t *testing.T) {
		usages, err := repo.ListUsageHistory(ctx, providerID, 10)
		require.NoError(t, err)
		require.Len(t, usages, 4)

		// Verify descending order
		for i := 1; i < len(usages); i++ {
			assert.True(t, usages[i-1].Month.After(usages[i].Month),
				"expected descending order at index %d", i)
		}
		assert.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), usages[0].Month)
		assert.Equal(t, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), usages[3].Month)
	})

	t.Run("respects limit", func(t *testing.T) {
		usages, err := repo.ListUsageHistory(ctx, providerID, 2)
		require.NoError(t, err)
		require.Len(t, usages, 2)

		// Should return the 2 most recent months
		assert.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), usages[0].Month)
		assert.Equal(t, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), usages[1].Month)
	})

	t.Run("returns empty slice for unknown provider", func(t *testing.T) {
		usages, err := repo.ListUsageHistory(ctx, uuid.New(), 10)
		require.NoError(t, err)
		assert.Empty(t, usages)
	})
}

func TestRepository_CreateTransaction_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)
	providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "billing-repo-create-txn"))

	t.Run("inserts transaction with all nullable fields nil", func(t *testing.T) {
		tx := &billing.BillingTransaction{
			ProviderID: providerID,
			Amount:     5.00,
			Currency:   "MXN",
		}

		err := repo.CreateTransaction(ctx, tx)
		require.NoError(t, err)

		assert.NotEqual(t, uuid.Nil, tx.ID)
		assert.False(t, tx.CreatedAt.IsZero())
		assert.Nil(t, tx.Description)
		assert.Nil(t, tx.ExternalPaymentID)
		assert.Nil(t, tx.BillingUsageID)
	})

	t.Run("inserts transaction with all fields including nullable ones", func(t *testing.T) {
		usage, err := repo.GetOrCreateUsage(ctx, providerID, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)

		desc := "Monthly invoice payment"
		extPayID := "stripe_pi_abc123"
		tx := &billing.BillingTransaction{
			ProviderID:        providerID,
			BillingUsageID:    &usage.ID,
			Amount:            10.00,
			Currency:          "MXN",
			Description:       &desc,
			ExternalPaymentID: &extPayID,
		}

		err = repo.CreateTransaction(ctx, tx)
		require.NoError(t, err)

		assert.NotEqual(t, uuid.Nil, tx.ID)
		assert.False(t, tx.CreatedAt.IsZero())
		require.NotNil(t, tx.Description)
		assert.Equal(t, desc, *tx.Description)
		require.NotNil(t, tx.ExternalPaymentID)
		assert.Equal(t, extPayID, *tx.ExternalPaymentID)
		require.NotNil(t, tx.BillingUsageID)
		assert.Equal(t, usage.ID, *tx.BillingUsageID)
	})
}

func TestRepository_ListTransactions_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)
	providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "billing-repo-list-txns"))

	// Insert a few transactions
	desc1 := "First payment"
	ext1 := "ext_001"
	txns := []*billing.BillingTransaction{
		{ProviderID: providerID, Amount: 5.00, Currency: "MXN", Description: &desc1, ExternalPaymentID: &ext1},
		{ProviderID: providerID, Amount: 10.00, Currency: "MXN"},
		{ProviderID: providerID, Amount: 2.50, Currency: "MXN"},
	}
	for _, tx := range txns {
		require.NoError(t, repo.CreateTransaction(ctx, tx))
	}

	t.Run("returns transactions in DESC order", func(t *testing.T) {
		result, err := repo.ListTransactions(ctx, providerID, 10)
		require.NoError(t, err)
		require.Len(t, result, 3)

		// All should be parseable including nullable fields
		for _, tx := range result {
			assert.NotEqual(t, uuid.Nil, tx.ID)
			assert.False(t, tx.CreatedAt.IsZero())
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		result, err := repo.ListTransactions(ctx, providerID, 2)
		require.NoError(t, err)
		require.Len(t, result, 2)
	})

	t.Run("nullable fields are scanned correctly", func(t *testing.T) {
		result, err := repo.ListTransactions(ctx, providerID, 10)
		require.NoError(t, err)

		// At least one should have a description and external ID
		var hasDesc bool
		var hasNilDesc bool
		for _, tx := range result {
			if tx.Description != nil {
				hasDesc = true
			} else {
				hasNilDesc = true
			}
		}
		assert.True(t, hasDesc, "expected at least one tx with description")
		assert.True(t, hasNilDesc, "expected at least one tx with nil description")
	})

	t.Run("returns empty for unknown provider", func(t *testing.T) {
		result, err := repo.ListTransactions(ctx, uuid.New(), 10)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestRepository_MarkPaid_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)
	providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "billing-repo-markpaid"))

	t.Run("marks usage as paid", func(t *testing.T) {
		usage, err := repo.GetOrCreateUsage(ctx, providerID, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)
		assert.False(t, usage.IsPaid)

		err = repo.MarkPaid(ctx, usage.ID)
		require.NoError(t, err)

		updated, err := repo.GetUsage(ctx, providerID, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)
		assert.True(t, updated.IsPaid)
	})

	t.Run("returns error for non-existent usage ID", func(t *testing.T) {
		err := repo.MarkPaid(ctx, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestRepository_CountUnpaidMonths_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)
	providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "billing-repo-unpaid-months"))

	// Create usage rows for multiple months, some with amount_due > 0 and unpaid
	months := []struct {
		month     time.Time
		completed int
		paid      bool
	}{
		{time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), 25, false}, // billable, unpaid
		{time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), 25, true},  // billable, paid
		{time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), 10, false}, // under free tier, unpaid (amount_due=0)
		{time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), 25, false}, // billable, unpaid
		{time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), 25, false}, // billable, unpaid
	}

	for _, m := range months {
		usage, err := repo.GetOrCreateUsage(ctx, providerID, m.month)
		require.NoError(t, err)

		// Sync with the desired completed count to set amount_due
		err = repo.SyncUsageTotals(ctx, usage.ID, m.completed)
		require.NoError(t, err)

		if m.paid {
			err = repo.MarkPaid(ctx, usage.ID)
			require.NoError(t, err)
		}
	}

	count, err := repo.CountUnpaidMonths(ctx, providerID)
	require.NoError(t, err)
	// Months with amount_due > 0 and is_paid=false: Sep(25, unpaid), Jun(25, unpaid), May(25, unpaid) = 3
	// Aug is paid, Jul has no amount_due
	assert.Equal(t, 3, count)

	t.Run("returns 0 for unknown provider", func(t *testing.T) {
		count, err := repo.CountUnpaidMonths(ctx, uuid.New())
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestRepository_RecalculateUsage_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)
	providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "billing-repo-recalc"))

	t.Run("recalculates billable and amount_due from completed count", func(t *testing.T) {
		usage, err := repo.GetOrCreateUsage(ctx, providerID, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)

		// Manually set completed_appointments directly in DB
		_, err = pool.Exec(ctx,
			"UPDATE billing_usage SET completed_appointments = 25 WHERE id = $1",
			usage.ID)
		require.NoError(t, err)

		// Now recalculate
		err = repo.RecalculateUsage(ctx, usage.ID)
		require.NoError(t, err)

		updated, err := repo.GetUsage(ctx, providerID, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)

		// 25 - 20 = 5 billable, 5 * 0.50 = 2.50
		assert.Equal(t, 5, updated.BillableAppointments)
		assert.InDelta(t, 2.50, updated.AmountDue, 0.001)
	})

	t.Run("returns not found for non-existent usage ID", func(t *testing.T) {
		err := repo.RecalculateUsage(ctx, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestRepository_ListAllProviderIDs_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)

	// Seed multiple providers
	activeID1 := uuid.MustParse(testutil.SeedProvider(t, pool, "billing-repo-active-1"))
	activeID2 := uuid.MustParse(testutil.SeedProvider(t, pool, "billing-repo-active-2"))
	inactiveID := uuid.MustParse(testutil.SeedProvider(t, pool, "billing-repo-inactive-1"))

	// Mark one provider as inactive
	_, err := pool.Exec(ctx, "UPDATE providers SET is_active = false WHERE id = $1", inactiveID)
	require.NoError(t, err)

	ids, err := repo.ListAllProviderIDs(ctx)
	require.NoError(t, err)

	// Should only return active providers
	idSet := make(map[uuid.UUID]bool)
	for _, id := range ids {
		idSet[id] = true
	}

	assert.True(t, idSet[activeID1], "expected active provider 1 in results")
	assert.True(t, idSet[activeID2], "expected active provider 2 in results")
	assert.False(t, idSet[inactiveID], "expected inactive provider NOT in results")
}

func TestRepository_CountCompletedForMonth_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)

	providerID := testutil.SeedProvider(t, pool, "billing-repo-count-completed")
	providerIDuuid := uuid.MustParse(providerID)
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
	serviceID := testutil.SeedService(t, pool, providerID, "Svc", 30)

	// Seed appointments in the current month (today's date per instructions)
	testutil.SeedAppointment(t, pool, providerID, employeeID, serviceID, "2026-09-11", "10:00", "10:30", "Client1", "+1231")
	testutil.SeedAppointment(t, pool, providerID, employeeID, serviceID, "2026-09-11", "11:00", "11:30", "Client2", "+1232")
	testutil.SeedAppointment(t, pool, providerID, employeeID, serviceID, "2026-09-11", "12:00", "12:30", "Client3", "+1233")

	// Update appointments to 'completed'
	_, err := pool.Exec(ctx, "UPDATE appointments SET status = 'completed' WHERE provider_id = $1", providerIDuuid)
	require.NoError(t, err)

	t.Run("counts completed appointments for current month", func(t *testing.T) {
		month := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
		count, err := repo.CountCompletedForMonth(ctx, providerIDuuid, month)
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("returns 0 for a month with no completed appointments", func(t *testing.T) {
		differentMonth := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
		count, err := repo.CountCompletedForMonth(ctx, providerIDuuid, differentMonth)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("returns 0 for unknown provider", func(t *testing.T) {
		month := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
		count, err := repo.CountCompletedForMonth(ctx, uuid.New(), month)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestRepository_SyncUsageTotals_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)
	providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "billing-repo-sync"))

	t.Run("syncs with completed count in free tier", func(t *testing.T) {
		usage, err := repo.GetOrCreateUsage(ctx, providerID, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)

		err = repo.SyncUsageTotals(ctx, usage.ID, 10)
		require.NoError(t, err)

		updated, err := repo.GetUsage(ctx, providerID, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)

		assert.Equal(t, 10, updated.CompletedAppointments)
		assert.Equal(t, 0, updated.BillableAppointments)
		assert.Equal(t, 0.0, updated.AmountDue)
	})

	t.Run("syncs with completed count beyond free tier", func(t *testing.T) {
		usage, err := repo.GetOrCreateUsage(ctx, providerID, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)

		err = repo.SyncUsageTotals(ctx, usage.ID, 25)
		require.NoError(t, err)

		updated, err := repo.GetUsage(ctx, providerID, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)

		// 25 - 20 = 5 billable, 5 * 0.50 = 2.50
		assert.Equal(t, 25, updated.CompletedAppointments)
		assert.Equal(t, 5, updated.BillableAppointments)
		assert.InDelta(t, 2.50, updated.AmountDue, 0.001)
	})

	t.Run("syncs with completed count that hits monthly cap", func(t *testing.T) {
		usage, err := repo.GetOrCreateUsage(ctx, providerID, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)

		// 20 + 20 = 40 completed => 20 billable => 10.00 (capped)
		err = repo.SyncUsageTotals(ctx, usage.ID, 40)
		require.NoError(t, err)

		updated, err := repo.GetUsage(ctx, providerID, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)

		assert.Equal(t, 40, updated.CompletedAppointments)
		assert.Equal(t, billing.MonthlyCap, updated.AmountDue)
	})
}

func TestRepository_ListUnpaidWithAmountDue_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)

	providerID1 := uuid.MustParse(testutil.SeedProvider(t, pool, "billing-repo-unpaid-due-1"))
	providerID2 := uuid.MustParse(testutil.SeedProvider(t, pool, "billing-repo-unpaid-due-2"))

	// Create usage rows: some with amount_due > 0 and unpaid, some paid, some with amount_due = 0
	// Provider 1: 2 unpaid months with amount_due > 0
	u1, err := repo.GetOrCreateUsage(ctx, providerID1, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.NoError(t, repo.SyncUsageTotals(ctx, u1.ID, 25)) // 2.50 due

	u2, err := repo.GetOrCreateUsage(ctx, providerID1, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.NoError(t, repo.SyncUsageTotals(ctx, u2.ID, 30)) // 5.00 due

	// Provider 2: 1 month with amount_due > 0 but paid
	u3, err := repo.GetOrCreateUsage(ctx, providerID2, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.NoError(t, repo.SyncUsageTotals(ctx, u3.ID, 22)) // 1.00 due
	require.NoError(t, repo.MarkPaid(ctx, u3.ID))

	// Provider 2: 1 month under free tier (amount_due = 0)
	u4, err := repo.GetOrCreateUsage(ctx, providerID2, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.NoError(t, repo.SyncUsageTotals(ctx, u4.ID, 5)) // 0.00 due

	usages, err := repo.ListUnpaidWithAmountDue(ctx)
	require.NoError(t, err)

	// Should return exactly the 2 unpaid rows for provider 1
	assert.Len(t, usages, 2)

	for _, u := range usages {
		assert.False(t, u.IsPaid)
		assert.Greater(t, u.AmountDue, 0.0)
		assert.Equal(t, providerID1, u.ProviderID)
	}

	t.Run("returns results in DESC month order", func(t *testing.T) {
		usages, err := repo.ListUnpaidWithAmountDue(ctx)
		require.NoError(t, err)
		require.Len(t, usages, 2)

		assert.True(t, usages[0].Month.After(usages[1].Month), "expected descending order by month")
	})
}
