package employee

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles employee data access.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new employee repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create inserts a new employee.
func (r *Repository) Create(ctx context.Context, e *Employee) error {
	query := `
		INSERT INTO employees (provider_id, name, phone, role, email, password_hash)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, is_active, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		e.ProviderID, e.Name, e.Phone, e.Role, e.Email, e.PasswordHash,
	).Scan(&e.ID, &e.IsActive, &e.CreatedAt, &e.UpdatedAt)
}

// GetByID retrieves an employee by ID.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Employee, error) {
	e := &Employee{}
	query := `
		SELECT id, provider_id, name, phone, role, email, password_hash,
		       is_active, created_at, updated_at
		FROM employees WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&e.ID, &e.ProviderID, &e.Name, &e.Phone, &e.Role,
		&e.Email, &e.PasswordHash, &e.IsActive, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("employee.repository: not found")
		}
		return nil, fmt.Errorf("employee.repository: get by id: %w", err)
	}
	return e, nil
}

// ListByProvider returns all employees for a provider.
func (r *Repository) ListByProvider(ctx context.Context, providerID uuid.UUID) ([]Employee, error) {
	query := `
		SELECT id, provider_id, name, phone, role, email, password_hash,
		       is_active, created_at, updated_at
		FROM employees
		WHERE provider_id = $1
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, providerID)
	if err != nil {
		return nil, fmt.Errorf("employee.repository: list: %w", err)
	}
	defer rows.Close()

	var employees []Employee
	for rows.Next() {
		var e Employee
		if err := rows.Scan(
			&e.ID, &e.ProviderID, &e.Name, &e.Phone, &e.Role,
			&e.Email, &e.PasswordHash, &e.IsActive, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("employee.repository: scan: %w", err)
		}
		employees = append(employees, e)
	}
	return employees, rows.Err()
}

// Update modifies an employee's fields.
func (r *Repository) Update(ctx context.Context, e *Employee) error {
	query := `
		UPDATE employees
		SET name = $2, phone = $3, is_active = $4
		WHERE id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query, e.ID, e.Name, e.Phone, e.IsActive).Scan(&e.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("employee.repository: not found")
		}
		return fmt.Errorf("employee.repository: update: %w", err)
	}
	return nil
}

// Delete removes an employee (hard delete via CASCADE).
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM employees WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("employee.repository: delete: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("employee.repository: not found")
	}
	return nil
}
