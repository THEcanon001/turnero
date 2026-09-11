// Package testutil provides shared test helpers for integration tests.
// It starts a real PostgreSQL container via testcontainers and applies migrations.
package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/THEcanon001/turnero/internal/platform/database"
)

const (
	testDBName = "turnero_test"
	testDBUser = "test_user"
	testDBPass = "test_pass"
)

// StartPostgres spins up a real PostgreSQL 16 container, applies all migrations,
// and returns a connected pgxpool.Pool. Call the returned cleanup function in defer.
// Callers should guard with:
//
//	if testing.Short() { t.Skip("skipping integration test") }
func StartPostgres(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(testDBName),
		postgres.WithUsername(testDBUser),
		postgres.WithPassword(testDBPass),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		t.Fatalf("testutil: failed to start postgres container: %v", err)
	}

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		t.Fatalf("testutil: failed to get connection string: %v", err)
	}

	// Apply migrations
	if err := database.Migrate(dsn); err != nil {
		_ = pgContainer.Terminate(ctx)
		t.Fatalf("testutil: failed to apply migrations: %v", err)
	}

	// Connect pool
	connCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := database.NewFromDSN(connCtx, dsn, 5, 1)
	if err != nil {
		_ = pgContainer.Terminate(ctx)
		t.Fatalf("testutil: failed to connect: %v", err)
	}

	cleanup := func() {
		pool.Close()
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("testutil: failed to terminate container: %v", err)
		}
	}

	return pool, cleanup
}

// SeedProvider inserts a provider for test setup and returns its UUID as string.
func SeedProvider(t *testing.T, pool *pgxpool.Pool, slug string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO providers (name, slug, phone, email, password_hash, type, timezone)
		VALUES ($1, $2, '+5491100000000', $3, '$2a$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ012', 'individual', 'America/Argentina/Buenos_Aires')
		RETURNING id`,
		"Test Provider "+slug, slug, slug+"@test.com",
	).Scan(&id)
	if err != nil {
		t.Fatalf("testutil: seed provider: %v", err)
	}
	return id
}

// SeedBusinessProvider inserts a business-type provider and returns its UUID as string.
func SeedBusinessProvider(t *testing.T, pool *pgxpool.Pool, slug string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO providers (name, slug, phone, email, password_hash, type, timezone, business_name)
		VALUES ($1, $2, '+5491100000000', $3, '$2a$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ012', 'business', 'America/Argentina/Buenos_Aires', $4)
		RETURNING id`,
		"Test Business "+slug, slug, slug+"@test.com", "Business "+slug,
	).Scan(&id)
	if err != nil {
		t.Fatalf("testutil: seed business provider: %v", err)
	}
	return id
}

// SeedEmployee inserts an employee under the given provider and returns its UUID as string.
func SeedEmployee(t *testing.T, pool *pgxpool.Pool, providerID, name string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO employees (provider_id, name, phone, role)
		VALUES ($1, $2, '+5491100000001', 'employee')
		RETURNING id`,
		providerID, name,
	).Scan(&id)
	if err != nil {
		t.Fatalf("testutil: seed employee: %v", err)
	}
	return id
}

// SeedAdminEmployee inserts an admin employee under the given provider and returns its UUID.
func SeedAdminEmployee(t *testing.T, pool *pgxpool.Pool, providerID, name, email string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO employees (provider_id, name, phone, role, email, password_hash)
		VALUES ($1, $2, '+5491100000002', 'admin', $3, '$2a$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ012')
		RETURNING id`,
		providerID, name, email,
	).Scan(&id)
	if err != nil {
		t.Fatalf("testutil: seed admin employee: %v", err)
	}
	return id
}

// SeedService inserts a service under the given provider and returns its UUID as string.
func SeedService(t *testing.T, pool *pgxpool.Pool, providerID, name string, durationMinutes int) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO services (provider_id, name, duration_minutes)
		VALUES ($1, $2, $3)
		RETURNING id`,
		providerID, name, durationMinutes,
	).Scan(&id)
	if err != nil {
		t.Fatalf("testutil: seed service: %v", err)
	}
	return id
}

// SeedSchedule inserts a weekly schedule for an employee.
func SeedSchedule(t *testing.T, pool *pgxpool.Pool, employeeID string, dayOfWeek int, startTime, endTime string, slotDuration int) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO schedules (employee_id, day_of_week, start_time, end_time, slot_duration_minutes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		employeeID, dayOfWeek, startTime, endTime, slotDuration,
	).Scan(&id)
	if err != nil {
		t.Fatalf("testutil: seed schedule: %v", err)
	}
	return id
}

// SeedAppointment inserts an appointment and returns its UUID.
func SeedAppointment(t *testing.T, pool *pgxpool.Pool, providerID, employeeID, serviceID, date, startTime, endTime, clientName, clientPhone string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO appointments (provider_id, employee_id, service_id, date, start_time, end_time, client_name, client_phone, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'confirmed')
		RETURNING id`,
		providerID, employeeID, serviceID, date, startTime, endTime, clientName, clientPhone,
	).Scan(&id)
	if err != nil {
		t.Fatalf("testutil: seed appointment: %v", err)
	}
	return id
}

// TruncateAll truncates all tables (useful between subtests sharing a container).
func TruncateAll(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		TRUNCATE audit_log, billing_transactions, billing_usage,
			push_tokens, refresh_tokens, invitation_codes,
			appointments, schedule_exceptions, schedules,
			employee_services, services, employees, providers
		CASCADE
	`)
	if err != nil {
		t.Fatalf("testutil: truncate all: %v", err)
	}
}
