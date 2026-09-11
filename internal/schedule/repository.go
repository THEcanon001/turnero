package schedule

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles schedule data access.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new schedule repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Upsert creates or updates a schedule for a given employee + day.
func (r *Repository) Upsert(ctx context.Context, s *Schedule) error {
	query := `
		INSERT INTO schedules (employee_id, day_of_week, start_time, end_time, slot_duration_minutes, break_after_slot_minutes)
		VALUES ($1, $2, $3::time, $4::time, $5, $6)
		ON CONFLICT (employee_id, day_of_week) DO UPDATE
		SET start_time = EXCLUDED.start_time,
		    end_time = EXCLUDED.end_time,
		    slot_duration_minutes = EXCLUDED.slot_duration_minutes,
		    break_after_slot_minutes = EXCLUDED.break_after_slot_minutes,
		    is_active = true
		RETURNING id, is_active, created_at`

	return r.pool.QueryRow(ctx, query,
		s.EmployeeID, s.DayOfWeek, s.StartTime, s.EndTime,
		s.SlotDurationMinutes, s.BreakAfterSlotMinutes,
	).Scan(&s.ID, &s.IsActive, &s.CreatedAt)
}

// ListByEmployee returns all schedules for an employee.
func (r *Repository) ListByEmployee(ctx context.Context, employeeID uuid.UUID) ([]Schedule, error) {
	query := `
		SELECT id, employee_id, day_of_week, start_time::text, end_time::text,
		       slot_duration_minutes, break_after_slot_minutes, is_active, created_at
		FROM schedules
		WHERE employee_id = $1
		ORDER BY day_of_week ASC`

	rows, err := r.pool.Query(ctx, query, employeeID)
	if err != nil {
		return nil, fmt.Errorf("schedule.repository: list: %w", err)
	}
	defer rows.Close()

	var schedules []Schedule
	for rows.Next() {
		var s Schedule
		if err := rows.Scan(
			&s.ID, &s.EmployeeID, &s.DayOfWeek, &s.StartTime, &s.EndTime,
			&s.SlotDurationMinutes, &s.BreakAfterSlotMinutes, &s.IsActive, &s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("schedule.repository: scan: %w", err)
		}
		schedules = append(schedules, s)
	}
	return schedules, rows.Err()
}

// Delete removes a schedule entry.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM schedules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("schedule.repository: delete: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("schedule.repository: not found")
	}
	return nil
}

// CreateException adds a schedule exception (day off or special hours).
func (r *Repository) CreateException(ctx context.Context, e *ScheduleException) error {
	query := `
		INSERT INTO schedule_exceptions (employee_id, date, is_available, start_time, end_time, reason)
		VALUES ($1, $2::date, $3, $4::time, $5::time, $6)
		ON CONFLICT (employee_id, date) DO UPDATE
		SET is_available = EXCLUDED.is_available,
		    start_time = EXCLUDED.start_time,
		    end_time = EXCLUDED.end_time,
		    reason = EXCLUDED.reason
		RETURNING id, created_at`

	return r.pool.QueryRow(ctx, query,
		e.EmployeeID, e.Date, e.IsAvailable,
		e.StartTime, e.EndTime, e.Reason,
	).Scan(&e.ID, &e.CreatedAt)
}

// ListExceptions returns schedule exceptions for an employee within a date range.
func (r *Repository) ListExceptions(ctx context.Context, employeeID uuid.UUID, from, to string) ([]ScheduleException, error) {
	query := `
		SELECT id, employee_id, date::text, is_available,
		       start_time::text, end_time::text, reason, created_at
		FROM schedule_exceptions
		WHERE employee_id = $1 AND date BETWEEN $2::date AND $3::date
		ORDER BY date ASC`

	rows, err := r.pool.Query(ctx, query, employeeID, from, to)
	if err != nil {
		return nil, fmt.Errorf("schedule.repository: list exceptions: %w", err)
	}
	defer rows.Close()

	var exceptions []ScheduleException
	for rows.Next() {
		var e ScheduleException
		if err := rows.Scan(
			&e.ID, &e.EmployeeID, &e.Date, &e.IsAvailable,
			&e.StartTime, &e.EndTime, &e.Reason, &e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("schedule.repository: scan exception: %w", err)
		}
		exceptions = append(exceptions, e)
	}
	return exceptions, rows.Err()
}

// GetExceptionByID retrieves a schedule exception by ID.
func (r *Repository) GetExceptionByID(ctx context.Context, id uuid.UUID) (*ScheduleException, error) {
	e := &ScheduleException{}
	query := `
		SELECT id, employee_id, date::text, is_available,
		       start_time::text, end_time::text, reason, created_at
		FROM schedule_exceptions WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&e.ID, &e.EmployeeID, &e.Date, &e.IsAvailable,
		&e.StartTime, &e.EndTime, &e.Reason, &e.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("schedule.repository: exception not found")
		}
		return nil, fmt.Errorf("schedule.repository: get exception: %w", err)
	}
	return e, nil
}

// DeleteException removes a schedule exception.
func (r *Repository) DeleteException(ctx context.Context, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM schedule_exceptions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("schedule.repository: delete exception: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("schedule.repository: exception not found")
	}
	return nil
}
