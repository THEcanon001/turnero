package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CreateService inserts a new service for a provider.
func (r *Repository) CreateService(ctx context.Context, s *Service) error {
	query := `
		INSERT INTO services (provider_id, name, description, duration_minutes)
		VALUES ($1, $2, $3, $4)
		RETURNING id, is_active, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		s.ProviderID, s.Name, s.Description, s.DurationMinutes,
	).Scan(&s.ID, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
}

// GetServiceByID retrieves a service by ID.
func (r *Repository) GetServiceByID(ctx context.Context, id uuid.UUID) (*Service, error) {
	s := &Service{}
	query := `
		SELECT id, provider_id, name, description, duration_minutes,
		       is_active, created_at, updated_at
		FROM services WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.ProviderID, &s.Name, &s.Description,
		&s.DurationMinutes, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("provider.repository: service not found")
		}
		return nil, fmt.Errorf("provider.repository: get service: %w", err)
	}
	return s, nil
}

// ListServices returns all services for a provider.
func (r *Repository) ListServices(ctx context.Context, providerID uuid.UUID) ([]Service, error) {
	query := `
		SELECT id, provider_id, name, description, duration_minutes,
		       is_active, created_at, updated_at
		FROM services
		WHERE provider_id = $1
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, providerID)
	if err != nil {
		return nil, fmt.Errorf("provider.repository: list services: %w", err)
	}
	defer rows.Close()

	var services []Service
	for rows.Next() {
		var s Service
		if err := rows.Scan(
			&s.ID, &s.ProviderID, &s.Name, &s.Description,
			&s.DurationMinutes, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("provider.repository: scan service: %w", err)
		}
		services = append(services, s)
	}
	return services, rows.Err()
}

// UpdateService modifies a service.
func (r *Repository) UpdateService(ctx context.Context, s *Service) error {
	query := `
		UPDATE services
		SET name = $2, description = $3, duration_minutes = $4, is_active = $5
		WHERE id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		s.ID, s.Name, s.Description, s.DurationMinutes, s.IsActive,
	).Scan(&s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("provider.repository: service not found")
		}
		return fmt.Errorf("provider.repository: update service: %w", err)
	}
	return nil
}

// DeleteService removes a service.
func (r *Repository) DeleteService(ctx context.Context, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM services WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("provider.repository: delete service: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("provider.repository: service not found")
	}
	return nil
}
