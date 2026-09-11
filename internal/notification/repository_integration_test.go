package notification_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/THEcanon001/turnero/internal/notification"
	"github.com/THEcanon001/turnero/internal/testutil"
)

func TestRepository_RegisterToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := notification.NewRepository(pool)
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "notif-repo")

	t.Run("registers token for provider", func(t *testing.T) {
		pid := uuid.MustParse(providerID)
		pt := &notification.PushToken{
			ProviderID: &pid,
			Token:      "expo-token-123",
			Platform:   "android",
		}

		err := repo.RegisterToken(ctx, pt)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, pt.ID)
		assert.True(t, pt.IsActive)
		assert.False(t, pt.CreatedAt.IsZero())
		assert.False(t, pt.UpdatedAt.IsZero())
	})

	t.Run("upsert updates existing token with new provider", func(t *testing.T) {
		providerID2 := testutil.SeedProvider(t, pool, "notif-repo-2")

		pid1 := uuid.MustParse(providerID)
		pt := &notification.PushToken{
			ProviderID: &pid1,
			Token:      "expo-token-upsert",
			Platform:   "ios",
		}
		err := repo.RegisterToken(ctx, pt)
		require.NoError(t, err)
		originalID := pt.ID

		// Register same token with a different provider — should upsert.
		pid2 := uuid.MustParse(providerID2)
		pt2 := &notification.PushToken{
			ProviderID: &pid2,
			Token:      "expo-token-upsert",
			Platform:   "ios",
		}
		err = repo.RegisterToken(ctx, pt2)
		require.NoError(t, err)

		// ID must be the same row (conflict on token).
		assert.Equal(t, originalID, pt2.ID)
		assert.True(t, pt2.IsActive)
		// Provider updated to the new one.
		require.NotNil(t, pt2.ProviderID)
		assert.Equal(t, pid2, *pt2.ProviderID)
	})

	t.Run("registers token for employee", func(t *testing.T) {
		employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
		eid := uuid.MustParse(employeeID)
		pt := &notification.PushToken{
			EmployeeID: &eid,
			Token:      "expo-token-emp-register",
			Platform:   "web",
		}

		err := repo.RegisterToken(ctx, pt)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, pt.ID)
		assert.True(t, pt.IsActive)
	})
}

func TestRepository_DeactivateToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := notification.NewRepository(pool)
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "notif-deactivate")

	t.Run("deactivates an existing active token", func(t *testing.T) {
		pid := uuid.MustParse(providerID)
		pt := &notification.PushToken{
			ProviderID: &pid,
			Token:      "expo-token-to-deactivate",
			Platform:   "android",
		}
		require.NoError(t, repo.RegisterToken(ctx, pt))

		// Confirm it's active before deactivation.
		tokens, err := repo.GetActiveTokensByProvider(ctx, pid)
		require.NoError(t, err)
		assert.Len(t, tokens, 1)

		// Deactivate.
		err = repo.DeactivateToken(ctx, "expo-token-to-deactivate")
		require.NoError(t, err)

		// Must no longer appear in active query.
		tokens, err = repo.GetActiveTokensByProvider(ctx, pid)
		require.NoError(t, err)
		assert.Empty(t, tokens)
	})

	t.Run("deactivating a non-existent token returns no error", func(t *testing.T) {
		err := repo.DeactivateToken(ctx, "token-that-does-not-exist")
		require.NoError(t, err)
	})
}

func TestRepository_GetActiveTokensByProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := notification.NewRepository(pool)
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "notif-by-provider")
	pid := uuid.MustParse(providerID)

	t.Run("returns all active tokens for provider", func(t *testing.T) {
		tokens := []string{"tok-prov-1", "tok-prov-2", "tok-prov-3"}
		for _, tok := range tokens {
			pt := &notification.PushToken{
				ProviderID: &pid,
				Token:      tok,
				Platform:   "android",
			}
			require.NoError(t, repo.RegisterToken(ctx, pt))
		}

		result, err := repo.GetActiveTokensByProvider(ctx, pid)
		require.NoError(t, err)
		assert.Len(t, result, 3)

		for _, r := range result {
			assert.True(t, r.IsActive)
			assert.Equal(t, pid, *r.ProviderID)
		}
	})

	t.Run("count decreases after deactivating one token", func(t *testing.T) {
		providerID2 := testutil.SeedProvider(t, pool, "notif-by-provider-2")
		pid2 := uuid.MustParse(providerID2)

		for _, tok := range []string{"tok-dec-1", "tok-dec-2"} {
			pt := &notification.PushToken{
				ProviderID: &pid2,
				Token:      tok,
				Platform:   "ios",
			}
			require.NoError(t, repo.RegisterToken(ctx, pt))
		}

		before, err := repo.GetActiveTokensByProvider(ctx, pid2)
		require.NoError(t, err)
		assert.Len(t, before, 2)

		require.NoError(t, repo.DeactivateToken(ctx, "tok-dec-1"))

		after, err := repo.GetActiveTokensByProvider(ctx, pid2)
		require.NoError(t, err)
		assert.Len(t, after, 1)
		assert.Equal(t, "tok-dec-2", after[0].Token)
	})

	t.Run("returns empty slice for provider with no tokens", func(t *testing.T) {
		unknownID := uuid.New()
		result, err := repo.GetActiveTokensByProvider(ctx, unknownID)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestRepository_GetActiveTokensByEmployee(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := notification.NewRepository(pool)
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "notif-by-employee")
	employeeID := testutil.SeedEmployee(t, pool, providerID, "Emp")
	eid := uuid.MustParse(employeeID)

	t.Run("returns active tokens for employee", func(t *testing.T) {
		pt := &notification.PushToken{
			EmployeeID: &eid,
			Token:      "expo-token-emp",
			Platform:   "ios",
		}
		require.NoError(t, repo.RegisterToken(ctx, pt))

		result, err := repo.GetActiveTokensByEmployee(ctx, eid)
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "expo-token-emp", result[0].Token)
		assert.True(t, result[0].IsActive)
		assert.Equal(t, eid, *result[0].EmployeeID)
	})

	t.Run("does not return deactivated token", func(t *testing.T) {
		employeeID2 := testutil.SeedEmployee(t, pool, providerID, "Emp2")
		eid2 := uuid.MustParse(employeeID2)

		pt := &notification.PushToken{
			EmployeeID: &eid2,
			Token:      "expo-token-emp-inactive",
			Platform:   "android",
		}
		require.NoError(t, repo.RegisterToken(ctx, pt))
		require.NoError(t, repo.DeactivateToken(ctx, "expo-token-emp-inactive"))

		result, err := repo.GetActiveTokensByEmployee(ctx, eid2)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("returns empty slice for employee with no tokens", func(t *testing.T) {
		unknownID := uuid.New()
		result, err := repo.GetActiveTokensByEmployee(ctx, unknownID)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("multiple tokens for same employee", func(t *testing.T) {
		employeeID3 := testutil.SeedEmployee(t, pool, providerID, "Emp3")
		eid3 := uuid.MustParse(employeeID3)

		for _, tok := range []string{"tok-multi-emp-1", "tok-multi-emp-2"} {
			pt := &notification.PushToken{
				EmployeeID: &eid3,
				Token:      tok,
				Platform:   "web",
			}
			require.NoError(t, repo.RegisterToken(ctx, pt))
		}

		result, err := repo.GetActiveTokensByEmployee(ctx, eid3)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})
}
