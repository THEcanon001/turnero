package appointment

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles appointment data access.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new appointment repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create inserts a new appointment. Returns a conflict error if the slot is taken.
func (r *Repository) Create(ctx context.Context, a *Appointment) error {
	query := `
		INSERT INTO appointments (provider_id, employee_id, service_id,
		    client_name, client_phone, date, start_time, end_time, notes)
		VALUES ($1, $2, $3, $4, $5, $6::date, $7::time, $8::time, $9)
		RETURNING id, status, reminder_sent, created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		a.ProviderID, a.EmployeeID, a.ServiceID,
		a.ClientName, a.ClientPhone, a.Date,
		a.StartTime, a.EndTime, a.Notes,
	).Scan(&a.ID, &a.Status, &a.ReminderSent, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("appointment.repository: slot already taken")
		}
		return fmt.Errorf("appointment.repository: create: %w", err)
	}
	return nil
}

// GetByID retrieves an appointment by ID.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Appointment, error) {
	a := &Appointment{}
	query := `
		SELECT id, provider_id, employee_id, service_id,
		       client_name, client_phone, date::text, start_time::text, end_time::text,
		       status, notes, cancellation_reason, reminder_sent, created_at, updated_at
		FROM appointments WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.ProviderID, &a.EmployeeID, &a.ServiceID,
		&a.ClientName, &a.ClientPhone, &a.Date, &a.StartTime, &a.EndTime,
		&a.Status, &a.Notes, &a.CancellationReason,
		&a.ReminderSent, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("appointment.repository: not found")
		}
		return nil, fmt.Errorf("appointment.repository: get by id: %w", err)
	}
	return a, nil
}

// List returns appointments matching the given filter.
func (r *Repository) List(ctx context.Context, f ListFilter) ([]Appointment, int, error) {
	// Build dynamic WHERE clause
	where := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	if f.ProviderID != nil {
		where += fmt.Sprintf(" AND provider_id = $%d", argIdx)
		args = append(args, *f.ProviderID)
		argIdx++
	}
	if f.EmployeeID != nil {
		where += fmt.Sprintf(" AND employee_id = $%d", argIdx)
		args = append(args, *f.EmployeeID)
		argIdx++
	}
	if f.Date != "" {
		where += fmt.Sprintf(" AND date = $%d::date", argIdx)
		args = append(args, f.Date)
		argIdx++
	}
	if f.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *f.Status)
		argIdx++
	}

	// Count
	countQuery := "SELECT COUNT(*) FROM appointments " + where
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("appointment.repository: count: %w", err)
	}

	// Paginated query
	if f.PerPage <= 0 {
		f.PerPage = 20
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	offset := (f.Page - 1) * f.PerPage

	query := fmt.Sprintf(`
		SELECT id, provider_id, employee_id, service_id,
		       client_name, client_phone, date::text, start_time::text, end_time::text,
		       status, notes, cancellation_reason, reminder_sent, created_at, updated_at
		FROM appointments %s
		ORDER BY date ASC, start_time ASC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)

	args = append(args, f.PerPage, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("appointment.repository: list: %w", err)
	}
	defer rows.Close()

	var appointments []Appointment
	for rows.Next() {
		var a Appointment
		if err := rows.Scan(
			&a.ID, &a.ProviderID, &a.EmployeeID, &a.ServiceID,
			&a.ClientName, &a.ClientPhone, &a.Date, &a.StartTime, &a.EndTime,
			&a.Status, &a.Notes, &a.CancellationReason,
			&a.ReminderSent, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("appointment.repository: scan: %w", err)
		}
		appointments = append(appointments, a)
	}
	return appointments, total, rows.Err()
}

// UpdateStatus changes the status of an appointment.
func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status, reason *string) error {
	query := `
		UPDATE appointments
		SET status = $2, cancellation_reason = $3
		WHERE id = $1
		RETURNING updated_at`

	var updatedAt any
	err := r.pool.QueryRow(ctx, query, id, status, reason).Scan(&updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("appointment.repository: not found")
		}
		return fmt.Errorf("appointment.repository: update status: %w", err)
	}
	return nil
}

// ListByEmployeeAndDate returns confirmed appointments for an employee on a given date.
// Used for slot availability calculation.
func (r *Repository) ListByEmployeeAndDate(ctx context.Context, employeeID uuid.UUID, date string) ([]Appointment, error) {
	query := `
		SELECT id, provider_id, employee_id, service_id,
		       client_name, client_phone, date::text, start_time::text, end_time::text,
		       status, notes, cancellation_reason, reminder_sent, created_at, updated_at
		FROM appointments
		WHERE employee_id = $1 AND date = $2::date AND status = 'confirmed'
		ORDER BY start_time ASC`

	rows, err := r.pool.Query(ctx, query, employeeID, date)
	if err != nil {
		return nil, fmt.Errorf("appointment.repository: list by employee+date: %w", err)
	}
	defer rows.Close()

	var appointments []Appointment
	for rows.Next() {
		var a Appointment
		if err := rows.Scan(
			&a.ID, &a.ProviderID, &a.EmployeeID, &a.ServiceID,
			&a.ClientName, &a.ClientPhone, &a.Date, &a.StartTime, &a.EndTime,
			&a.Status, &a.Notes, &a.CancellationReason,
			&a.ReminderSent, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("appointment.repository: scan: %w", err)
		}
		appointments = append(appointments, a)
	}
	return appointments, rows.Err()
}
