package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles refresh token data access.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new auth repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// StoreRefreshToken saves a hashed refresh token.
func (r *Repository) StoreRefreshToken(ctx context.Context, rt *RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (provider_id, employee_id, token_hash, expires_at, user_agent, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	return r.pool.QueryRow(ctx, query,
		rt.ProviderID, rt.EmployeeID, rt.TokenHash,
		rt.ExpiresAt, rt.UserAgent, rt.IPAddress,
	).Scan(&rt.ID, &rt.CreatedAt)
}

// GetByTokenHash retrieves a refresh token by its hash.
func (r *Repository) GetByTokenHash(ctx context.Context, hash string) (*RefreshToken, error) {
	rt := &RefreshToken{}
	query := `
		SELECT id, provider_id, employee_id, token_hash, expires_at,
		       revoked_at, replaced_by, user_agent, ip_address, created_at
		FROM refresh_tokens
		WHERE token_hash = $1 AND revoked_at IS NULL`

	err := r.pool.QueryRow(ctx, query, hash).Scan(
		&rt.ID, &rt.ProviderID, &rt.EmployeeID, &rt.TokenHash,
		&rt.ExpiresAt, &rt.RevokedAt, &rt.ReplacedBy,
		&rt.UserAgent, &rt.IPAddress, &rt.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("auth.repository: refresh token not found")
		}
		return nil, fmt.Errorf("auth.repository: get refresh token: %w", err)
	}
	return rt, nil
}

// RevokeToken marks a refresh token as revoked and optionally links it to the replacement.
func (r *Repository) RevokeToken(ctx context.Context, tokenID uuid.UUID, replacedBy *uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = $2, replaced_by = $3
		WHERE id = $1`

	now := time.Now()
	_, err := r.pool.Exec(ctx, query, tokenID, now, replacedBy)
	if err != nil {
		return fmt.Errorf("auth.repository: revoke token: %w", err)
	}
	return nil
}

// RevokeAllForProvider revokes all refresh tokens for a provider (e.g., on password change).
func (r *Repository) RevokeAllForProvider(ctx context.Context, providerID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE provider_id = $1 AND revoked_at IS NULL`

	_, err := r.pool.Exec(ctx, query, providerID)
	if err != nil {
		return fmt.Errorf("auth.repository: revoke all: %w", err)
	}
	return nil
}

// HashToken returns the SHA-256 hash of a token string.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
