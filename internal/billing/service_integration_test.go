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

func TestService_RecordCompletion_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)
	svc := billing.NewService(repo)

	t.Run("first completion is within free tier", func(t *testing.T) {
		providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "svc-record-free"))

		usage, exceeded, err := svc.RecordCompletion(ctx, providerID)
		require.NoError(t, err)
		require.NotNil(t, usage)

		assert.Equal(t, 1, usage.CompletedAppointments)
		assert.False(t, exceeded)
	})

	t.Run("21st completion exceeds free tier", func(t *testing.T) {
		providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "svc-record-exceed"))

		// Call 20 times (exactly at free tier — exceeded should still be false)
		for i := 0; i < billing.FreeTierLimit; i++ {
			usage, exceeded, err := svc.RecordCompletion(ctx, providerID)
			require.NoError(t, err)
			assert.False(t, exceeded, "should not exceed free tier at call %d", i+1)
			assert.Equal(t, i+1, usage.CompletedAppointments)
		}

		// 21st call should cross the free tier
		usage, exceeded, err := svc.RecordCompletion(ctx, providerID)
		require.NoError(t, err)
		require.NotNil(t, usage)

		assert.Equal(t, billing.FreeTierLimit+1, usage.CompletedAppointments)
		assert.True(t, exceeded, "expected exceededFreeTier=true on 21st call")
		assert.Equal(t, 1, usage.BillableAppointments)
		assert.InDelta(t, billing.PricePerAppointment, usage.AmountDue, 0.001)
	})
}

func TestService_IsRestricted_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)
	svc := billing.NewService(repo)

	t.Run("not restricted with fewer than GraceThreshold unpaid months", func(t *testing.T) {
		providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "svc-restrict-below"))

		// Create 2 unpaid months with amount_due > 0 (one below threshold)
		for i, month := range []time.Time{
			time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		} {
			usage, err := repo.GetOrCreateUsage(ctx, providerID, month)
			require.NoError(t, err, "month %d", i)
			require.NoError(t, repo.SyncUsageTotals(ctx, usage.ID, 25))
		}

		restricted, err := svc.IsRestricted(ctx, providerID)
		require.NoError(t, err)
		assert.False(t, restricted)
	})

	t.Run("restricted with exactly GraceThreshold unpaid months", func(t *testing.T) {
		providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "svc-restrict-exact"))

		// Create exactly 3 unpaid months with amount_due > 0
		months := []time.Time{
			time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		}
		for i, month := range months {
			usage, err := repo.GetOrCreateUsage(ctx, providerID, month)
			require.NoError(t, err, "month %d", i)
			require.NoError(t, repo.SyncUsageTotals(ctx, usage.ID, 25))
		}

		restricted, err := svc.IsRestricted(ctx, providerID)
		require.NoError(t, err)
		assert.True(t, restricted, "expected restriction with %d unpaid months", billing.GraceThreshold)
	})

	t.Run("not restricted when months are paid", func(t *testing.T) {
		providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "svc-restrict-paid"))

		// Create 3 months with amount_due > 0 but mark them all paid
		months := []time.Time{
			time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		}
		for i, month := range months {
			usage, err := repo.GetOrCreateUsage(ctx, providerID, month)
			require.NoError(t, err, "month %d", i)
			require.NoError(t, repo.SyncUsageTotals(ctx, usage.ID, 25))
			require.NoError(t, repo.MarkPaid(ctx, usage.ID))
		}

		restricted, err := svc.IsRestricted(ctx, providerID)
		require.NoError(t, err)
		assert.False(t, restricted)
	})

	t.Run("not restricted when amount_due is zero even if unpaid", func(t *testing.T) {
		providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "svc-restrict-zero-due"))

		// Create 3+ months all under free tier (amount_due = 0)
		months := []time.Time{
			time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		}
		for i, month := range months {
			usage, err := repo.GetOrCreateUsage(ctx, providerID, month)
			require.NoError(t, err, "month %d", i)
			require.NoError(t, repo.SyncUsageTotals(ctx, usage.ID, 5)) // under free tier
		}

		restricted, err := svc.IsRestricted(ctx, providerID)
		require.NoError(t, err)
		assert.False(t, restricted)
	})
}

func TestService_GetCurrentUsage_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	ctx := context.Background()
	repo := billing.NewRepository(pool)
	svc := billing.NewService(repo)

	t.Run("retrieves or creates current month usage", func(t *testing.T) {
		providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "svc-current-usage"))

		usage, err := svc.GetCurrentUsage(ctx, providerID)
		require.NoError(t, err)
		require.NotNil(t, usage)

		assert.Equal(t, providerID, usage.ProviderID)
		assert.Equal(t, billing.FreeTierLimit, usage.FreeTierLimit)

		// Month should be the first of the current month
		now := time.Now()
		expectedMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, expectedMonth, usage.Month)
	})

	t.Run("returns same row on repeated calls", func(t *testing.T) {
		providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "svc-current-usage-idem"))

		u1, err := svc.GetCurrentUsage(ctx, providerID)
		require.NoError(t, err)

		u2, err := svc.GetCurrentUsage(ctx, providerID)
		require.NoError(t, err)

		assert.Equal(t, u1.ID, u2.ID)
	})

	t.Run("reflects increments after RecordCompletion", func(t *testing.T) {
		providerID := uuid.MustParse(testutil.SeedProvider(t, pool, "svc-current-after-incr"))

		_, _, err := svc.RecordCompletion(ctx, providerID)
		require.NoError(t, err)
		_, _, err = svc.RecordCompletion(ctx, providerID)
		require.NoError(t, err)

		usage, err := svc.GetCurrentUsage(ctx, providerID)
		require.NoError(t, err)

		// GetCurrentUsage calls GetOrCreateUsage which doesn't decrement, so count should be 2
		assert.Equal(t, 2, usage.CompletedAppointments)
	})
}
