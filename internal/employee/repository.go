package employee

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

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

// GetByEmail retrieves an employee by email.
func (r *Repository) GetByEmail(ctx context.Context, email string) (*Employee, error) {
	e := &Employee{}
	query := `
		SELECT id, provider_id, name, phone, role, email, password_hash,
		       is_active, created_at, updated_at
		FROM employees WHERE email = $1`

	err := r.pool.QueryRow(ctx, query, email).Scan(
		&e.ID, &e.ProviderID, &e.Name, &e.Phone, &e.Role,
		&e.Email, &e.PasswordHash, &e.IsActive, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("employee.repository: not found")
		}
		return nil, fmt.Errorf("employee.repository: get by email: %w", err)
	}
	return e, nil
}

// CreateInvitation generates and stores an invitation code.
func (r *Repository) CreateInvitation(ctx context.Context, providerID uuid.UUID, employeeName string, ttl time.Duration) (*InvitationCode, error) {
	code, err := generateCode(8)
	if err != nil {
		return nil, fmt.Errorf("employee.repository: generate code: %w", err)
	}

	inv := &InvitationCode{
		ProviderID:   providerID,
		Code:         code,
		EmployeeName: employeeName,
		ExpiresAt:    time.Now().Add(ttl),
	}

	query := `
		INSERT INTO invitation_codes (provider_id, code, employee_name, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	err = r.pool.QueryRow(ctx, query,
		inv.ProviderID, inv.Code, inv.EmployeeName, inv.ExpiresAt,
	).Scan(&inv.ID, &inv.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("employee.repository: create invitation: %w", err)
	}
	return inv, nil
}

// GetInvitationByCode retrieves a valid (unused, non-expired) invitation by code.
func (r *Repository) GetInvitationByCode(ctx context.Context, code string) (*InvitationCode, error) {
	inv := &InvitationCode{}
	query := `
		SELECT id, provider_id, code, employee_name, expires_at, used_at, used_by, created_at
		FROM invitation_codes
		WHERE code = $1 AND used_at IS NULL`

	err := r.pool.QueryRow(ctx, query, code).Scan(
		&inv.ID, &inv.ProviderID, &inv.Code, &inv.EmployeeName,
		&inv.ExpiresAt, &inv.UsedAt, &inv.UsedBy, &inv.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("employee.repository: invitation not found or already used")
		}
		return nil, fmt.Errorf("employee.repository: get invitation: %w", err)
	}

	if time.Now().After(inv.ExpiresAt) {
		return nil, fmt.Errorf("employee.repository: invitation expired")
	}

	return inv, nil
}

// MarkInvitationUsed marks an invitation code as used by an employee.
func (r *Repository) MarkInvitationUsed(ctx context.Context, invitationID, employeeID uuid.UUID) error {
	query := `UPDATE invitation_codes SET used_at = now(), used_by = $2 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, invitationID, employeeID)
	if err != nil {
		return fmt.Errorf("employee.repository: mark invitation used: %w", err)
	}
	return nil
}

// ListInvitations returns all invitation codes for a provider.
func (r *Repository) ListInvitations(ctx context.Context, providerID uuid.UUID) ([]InvitationCode, error) {
	query := `
		SELECT id, provider_id, code, employee_name, expires_at, used_at, used_by, created_at
		FROM invitation_codes
		WHERE provider_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, providerID)
	if err != nil {
		return nil, fmt.Errorf("employee.repository: list invitations: %w", err)
	}
	defer rows.Close()

	var invitations []InvitationCode
	for rows.Next() {
		var inv InvitationCode
		if err := rows.Scan(
			&inv.ID, &inv.ProviderID, &inv.Code, &inv.EmployeeName,
			&inv.ExpiresAt, &inv.UsedAt, &inv.UsedBy, &inv.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("employee.repository: scan invitation: %w", err)
		}
		invitations = append(invitations, inv)
	}
	return invitations, rows.Err()
}

// AssignServices replaces the employee's service assignments.
func (r *Repository) AssignServices(ctx context.Context, employeeID uuid.UUID, serviceIDs []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("employee.repository: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Remove existing
	if _, err := tx.Exec(ctx, `DELETE FROM employee_services WHERE employee_id = $1`, employeeID); err != nil {
		return fmt.Errorf("employee.repository: clear services: %w", err)
	}

	// Insert new
	for _, sid := range serviceIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO employee_services (employee_id, service_id) VALUES ($1, $2)`,
			employeeID, sid,
		); err != nil {
			return fmt.Errorf("employee.repository: assign service: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("employee.repository: commit: %w", err)
	}
	return nil
}

// ListServiceIDs returns the service IDs assigned to an employee.
func (r *Repository) ListServiceIDs(ctx context.Context, employeeID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT service_id FROM employee_services WHERE employee_id = $1`
	rows, err := r.pool.Query(ctx, query, employeeID)
	if err != nil {
		return nil, fmt.Errorf("employee.repository: list service ids: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("employee.repository: scan service id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// generateCode creates a random alphanumeric code of the given length.
func generateCode(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b)[:length], nil
}
