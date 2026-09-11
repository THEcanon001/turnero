package audit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles audit log data access.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new audit repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Log inserts a new audit log entry.
func (r *Repository) Log(ctx context.Context, entry *AuditEntry) error {
	var metadataJSON []byte
	var err error
	if entry.Metadata != nil {
		metadataJSON, err = json.Marshal(entry.Metadata)
		if err != nil {
			return fmt.Errorf("audit.repository: Log: marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO audit_log (actor_type, actor_id, resource_type, resource_id, action, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	err = r.pool.QueryRow(ctx, query,
		entry.ActorType, entry.ActorID, entry.ResourceType,
		entry.ResourceID, entry.Action, metadataJSON,
	).Scan(&entry.ID, &entry.CreatedAt)
	if err != nil {
		return fmt.Errorf("audit.repository: Log: %w", err)
	}
	return nil
}

// List queries audit log entries with filtering and pagination.
func (r *Repository) List(ctx context.Context, filter ListFilter) ([]AuditEntry, int, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 20
	}

	where := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	if filter.ActorType != "" {
		where += fmt.Sprintf(" AND actor_type = $%d", argIdx)
		args = append(args, filter.ActorType)
		argIdx++
	}
	if filter.ActorID != nil {
		where += fmt.Sprintf(" AND actor_id = $%d", argIdx)
		args = append(args, *filter.ActorID)
		argIdx++
	}
	if filter.ResourceType != "" {
		where += fmt.Sprintf(" AND resource_type = $%d", argIdx)
		args = append(args, filter.ResourceType)
		argIdx++
	}
	if filter.ResourceID != nil {
		where += fmt.Sprintf(" AND resource_id = $%d", argIdx)
		args = append(args, *filter.ResourceID)
		argIdx++
	}
	if filter.Action != "" {
		where += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, filter.Action)
		argIdx++
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM audit_log " + where
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("audit.repository: List: count: %w", err)
	}

	// Fetch page
	offset := (filter.Page - 1) * filter.PerPage
	selectQuery := fmt.Sprintf(`
		SELECT id, actor_type, actor_id, resource_type, resource_id, action, metadata, created_at
		FROM audit_log %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, filter.PerPage, offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("audit.repository: List: %w", err)
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var metadataJSON []byte
		if err := rows.Scan(
			&e.ID, &e.ActorType, &e.ActorID, &e.ResourceType,
			&e.ResourceID, &e.Action, &metadataJSON, &e.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("audit.repository: List: scan: %w", err)
		}
		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &e.Metadata); err != nil {
				return nil, 0, fmt.Errorf("audit.repository: List: unmarshal metadata: %w", err)
			}
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("audit.repository: List: rows: %w", err)
	}

	return entries, total, nil
}
