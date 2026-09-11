package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/THEcanon001/turnero/internal/platform/database"
)

func TestNewFromDSN_ConnectsToDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	dsn, cleanup := startPostgres(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewFromDSN(ctx, dsn, 5, 1)
	require.NoError(t, err)
	defer pool.Close()

	err = pool.Ping(ctx)
	assert.NoError(t, err)
}

func TestNewFromDSN_InvalidDSN(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := database.NewFromDSN(ctx, "postgres://bad:bad@localhost:1/bad?sslmode=disable", 5, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database:")
}

func TestMigrate_AppliesSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	dsn, cleanup := startPostgres(t)
	defer cleanup()

	// Run migrations
	err := database.Migrate(dsn)
	require.NoError(t, err)

	// Verify tables exist by connecting and querying
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewFromDSN(ctx, dsn, 5, 1)
	require.NoError(t, err)
	defer pool.Close()

	// Check that core tables exist
	tables := []string{
		"providers", "employees", "services", "employee_services",
		"schedules", "schedule_exceptions", "appointments",
		"invitation_codes", "billing_usage", "billing_transactions",
		"push_tokens", "refresh_tokens", "audit_log",
	}

	for _, table := range tables {
		var exists bool
		err := pool.QueryRow(ctx,
			"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)",
			table,
		).Scan(&exists)
		require.NoError(t, err, "error checking table %s", table)
		assert.True(t, exists, "table %s should exist after migration", table)
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	dsn, cleanup := startPostgres(t)
	defer cleanup()

	// Run migrations twice — should not error
	err := database.Migrate(dsn)
	require.NoError(t, err)

	err = database.Migrate(dsn)
	require.NoError(t, err)
}
