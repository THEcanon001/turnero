package notification

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles push token data access.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new notification repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// RegisterToken upserts a push token for a provider/employee.
func (r *Repository) RegisterToken(ctx context.Context, pt *PushToken) error {
	query := `
		INSERT INTO push_tokens (provider_id, employee_id, client_phone, token, platform)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (token) DO UPDATE
		SET is_active = true, provider_id = EXCLUDED.provider_id,
		    employee_id = EXCLUDED.employee_id, client_phone = EXCLUDED.client_phone
		RETURNING id, is_active, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		pt.ProviderID, pt.EmployeeID, pt.ClientPhone, pt.Token, pt.Platform,
	).Scan(&pt.ID, &pt.IsActive, &pt.CreatedAt, &pt.UpdatedAt)
}

// DeactivateToken marks a push token as inactive.
func (r *Repository) DeactivateToken(ctx context.Context, token string) error {
	query := `UPDATE push_tokens SET is_active = false WHERE token = $1`
	_, err := r.pool.Exec(ctx, query, token)
	if err != nil {
		return fmt.Errorf("notification.repository: deactivate: %w", err)
	}
	return nil
}

// GetActiveTokensByProvider returns all active push tokens for a provider.
func (r *Repository) GetActiveTokensByProvider(ctx context.Context, providerID uuid.UUID) ([]PushToken, error) {
	return r.queryTokens(ctx,
		`SELECT id, provider_id, employee_id, client_phone, token, platform, is_active, created_at, updated_at
		 FROM push_tokens WHERE provider_id = $1 AND is_active = true`, providerID)
}

// GetActiveTokensByEmployee returns all active push tokens for an employee.
func (r *Repository) GetActiveTokensByEmployee(ctx context.Context, employeeID uuid.UUID) ([]PushToken, error) {
	return r.queryTokens(ctx,
		`SELECT id, provider_id, employee_id, client_phone, token, platform, is_active, created_at, updated_at
		 FROM push_tokens WHERE employee_id = $1 AND is_active = true`, employeeID)
}

func (r *Repository) queryTokens(ctx context.Context, query string, args ...any) ([]PushToken, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("notification.repository: query tokens: %w", err)
	}
	defer rows.Close()

	var tokens []PushToken
	for rows.Next() {
		var t PushToken
		if err := rows.Scan(
			&t.ID, &t.ProviderID, &t.EmployeeID, &t.ClientPhone,
			&t.Token, &t.Platform, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("notification.repository: scan: %w", err)
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}
