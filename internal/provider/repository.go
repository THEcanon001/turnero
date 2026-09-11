package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles provider data access.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new provider repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create inserts a new provider and returns the created record.
func (r *Repository) Create(ctx context.Context, p *Provider) error {
	query := `
		INSERT INTO providers (type, name, slug, phone, address, timezone, business_name, email, password_hash, google_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, plan, is_active, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		p.Type, p.Name, p.Slug, p.Phone, p.Address,
		p.Timezone, p.BusinessName, p.Email, p.PasswordHash, p.GoogleID,
	).Scan(&p.ID, &p.Plan, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
}

// GetByID retrieves a provider by ID.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Provider, error) {
	p := &Provider{}
	query := `
		SELECT id, type, name, slug, phone, address, timezone, business_name,
		       email, password_hash, google_id, qr_image_path, plan, billing_cycle_start,
		       is_active, created_at, updated_at
		FROM providers WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Type, &p.Name, &p.Slug, &p.Phone, &p.Address,
		&p.Timezone, &p.BusinessName, &p.Email, &p.PasswordHash,
		&p.GoogleID, &p.QRImagePath, &p.Plan, &p.BillingCycleStart,
		&p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("provider.repository: not found")
		}
		return nil, fmt.Errorf("provider.repository: get by id: %w", err)
	}
	return p, nil
}

// GetByEmail retrieves a provider by email.
func (r *Repository) GetByEmail(ctx context.Context, email string) (*Provider, error) {
	p := &Provider{}
	query := `
		SELECT id, type, name, slug, phone, address, timezone, business_name,
		       email, password_hash, google_id, qr_image_path, plan, billing_cycle_start,
		       is_active, created_at, updated_at
		FROM providers WHERE email = $1`

	err := r.pool.QueryRow(ctx, query, email).Scan(
		&p.ID, &p.Type, &p.Name, &p.Slug, &p.Phone, &p.Address,
		&p.Timezone, &p.BusinessName, &p.Email, &p.PasswordHash,
		&p.GoogleID, &p.QRImagePath, &p.Plan, &p.BillingCycleStart,
		&p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("provider.repository: not found")
		}
		return nil, fmt.Errorf("provider.repository: get by email: %w", err)
	}
	return p, nil
}

// GetBySlug retrieves a provider by slug.
func (r *Repository) GetBySlug(ctx context.Context, slug string) (*Provider, error) {
	p := &Provider{}
	query := `
		SELECT id, type, name, slug, phone, address, timezone, business_name,
		       email, password_hash, google_id, qr_image_path, plan, billing_cycle_start,
		       is_active, created_at, updated_at
		FROM providers WHERE slug = $1`

	err := r.pool.QueryRow(ctx, query, slug).Scan(
		&p.ID, &p.Type, &p.Name, &p.Slug, &p.Phone, &p.Address,
		&p.Timezone, &p.BusinessName, &p.Email, &p.PasswordHash,
		&p.GoogleID, &p.QRImagePath, &p.Plan, &p.BillingCycleStart,
		&p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("provider.repository: not found")
		}
		return nil, fmt.Errorf("provider.repository: get by slug: %w", err)
	}
	return p, nil
}

// GetByGoogleID retrieves a provider by Google OAuth ID.
func (r *Repository) GetByGoogleID(ctx context.Context, googleID string) (*Provider, error) {
	p := &Provider{}
	query := `
		SELECT id, type, name, slug, phone, address, timezone, business_name,
		       email, password_hash, google_id, qr_image_path, plan, billing_cycle_start,
		       is_active, created_at, updated_at
		FROM providers WHERE google_id = $1`

	err := r.pool.QueryRow(ctx, query, googleID).Scan(
		&p.ID, &p.Type, &p.Name, &p.Slug, &p.Phone, &p.Address,
		&p.Timezone, &p.BusinessName, &p.Email, &p.PasswordHash,
		&p.GoogleID, &p.QRImagePath, &p.Plan, &p.BillingCycleStart,
		&p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("provider.repository: not found")
		}
		return nil, fmt.Errorf("provider.repository: get by google id: %w", err)
	}
	return p, nil
}

// LinkGoogleID associates a Google OAuth ID with an existing provider.
func (r *Repository) LinkGoogleID(ctx context.Context, providerID uuid.UUID, googleID string) error {
	query := `UPDATE providers SET google_id = $2 WHERE id = $1`
	ct, err := r.pool.Exec(ctx, query, providerID, googleID)
	if err != nil {
		return fmt.Errorf("provider.repository: link google id: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("provider.repository: not found")
	}
	return nil
}

// Update modifies an existing provider's profile fields.
func (r *Repository) Update(ctx context.Context, p *Provider) error {
	query := `
		UPDATE providers
		SET name = $2, phone = $3, address = $4, timezone = $5, business_name = $6
		WHERE id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		p.ID, p.Name, p.Phone, p.Address, p.Timezone, p.BusinessName,
	).Scan(&p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("provider.repository: not found")
		}
		return fmt.Errorf("provider.repository: update: %w", err)
	}
	return nil
}
