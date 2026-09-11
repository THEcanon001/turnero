package billing

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles billing data access.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new billing repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetOrCreateUsage upserts a billing_usage row for the given provider and month.
func (r *Repository) GetOrCreateUsage(ctx context.Context, providerID uuid.UUID, month time.Time) (*BillingUsage, error) {
	// Normalize month to first of month
	m := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)

	query := `
		INSERT INTO billing_usage (provider_id, month, free_tier_limit)
		VALUES ($1, $2, $3)
		ON CONFLICT (provider_id, month) DO UPDATE SET updated_at = now()
		RETURNING id, provider_id, month, completed_appointments, free_tier_limit,
				  billable_appointments, amount_due, is_paid, created_at, updated_at`

	var u BillingUsage
	err := r.pool.QueryRow(ctx, query, providerID, m, FreeTierLimit).Scan(
		&u.ID, &u.ProviderID, &u.Month, &u.CompletedAppointments, &u.FreeTierLimit,
		&u.BillableAppointments, &u.AmountDue, &u.IsPaid, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("billing.repository: GetOrCreateUsage: %w", err)
	}
	return &u, nil
}

// IncrementCompleted atomically increments the completed count for the current month
// and recalculates billable appointments and amount due.
func (r *Repository) IncrementCompleted(ctx context.Context, providerID uuid.UUID) (*BillingUsage, error) {
	m := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)

	// Ensure the row exists
	_, err := r.GetOrCreateUsage(ctx, providerID, m)
	if err != nil {
		return nil, fmt.Errorf("billing.repository: IncrementCompleted: %w", err)
	}

	query := `
		UPDATE billing_usage
		SET completed_appointments = completed_appointments + 1,
			billable_appointments = GREATEST(0, completed_appointments + 1 - free_tier_limit),
			amount_due = LEAST($1, GREATEST(0, completed_appointments + 1 - free_tier_limit) * $2),
			updated_at = now()
		WHERE provider_id = $3 AND month = $4
		RETURNING id, provider_id, month, completed_appointments, free_tier_limit,
				  billable_appointments, amount_due, is_paid, created_at, updated_at`

	var u BillingUsage
	err = r.pool.QueryRow(ctx, query, MonthlyCap, PricePerAppointment, providerID, m).Scan(
		&u.ID, &u.ProviderID, &u.Month, &u.CompletedAppointments, &u.FreeTierLimit,
		&u.BillableAppointments, &u.AmountDue, &u.IsPaid, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("billing.repository: IncrementCompleted: %w", err)
	}
	return &u, nil
}

// GetUsage returns the billing usage for a specific provider and month.
func (r *Repository) GetUsage(ctx context.Context, providerID uuid.UUID, month time.Time) (*BillingUsage, error) {
	m := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)

	query := `
		SELECT id, provider_id, month, completed_appointments, free_tier_limit,
			   billable_appointments, amount_due, is_paid, created_at, updated_at
		FROM billing_usage
		WHERE provider_id = $1 AND month = $2`

	var u BillingUsage
	err := r.pool.QueryRow(ctx, query, providerID, m).Scan(
		&u.ID, &u.ProviderID, &u.Month, &u.CompletedAppointments, &u.FreeTierLimit,
		&u.BillableAppointments, &u.AmountDue, &u.IsPaid, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("billing.repository: GetUsage: not found")
		}
		return nil, fmt.Errorf("billing.repository: GetUsage: %w", err)
	}
	return &u, nil
}

// GetCurrentUsage returns the billing usage for the current month.
func (r *Repository) GetCurrentUsage(ctx context.Context, providerID uuid.UUID) (*BillingUsage, error) {
	return r.GetOrCreateUsage(ctx, providerID, time.Now())
}

// ListUsageHistory returns the last N months of billing usage for a provider.
func (r *Repository) ListUsageHistory(ctx context.Context, providerID uuid.UUID, limit int) ([]BillingUsage, error) {
	query := `
		SELECT id, provider_id, month, completed_appointments, free_tier_limit,
			   billable_appointments, amount_due, is_paid, created_at, updated_at
		FROM billing_usage
		WHERE provider_id = $1
		ORDER BY month DESC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, query, providerID, limit)
	if err != nil {
		return nil, fmt.Errorf("billing.repository: ListUsageHistory: %w", err)
	}
	defer rows.Close()

	var usages []BillingUsage
	for rows.Next() {
		var u BillingUsage
		if err := rows.Scan(
			&u.ID, &u.ProviderID, &u.Month, &u.CompletedAppointments, &u.FreeTierLimit,
			&u.BillableAppointments, &u.AmountDue, &u.IsPaid, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("billing.repository: ListUsageHistory: %w", err)
		}
		usages = append(usages, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("billing.repository: ListUsageHistory: %w", err)
	}
	return usages, nil
}

// CreateTransaction inserts a new billing transaction.
func (r *Repository) CreateTransaction(ctx context.Context, tx *BillingTransaction) error {
	query := `
		INSERT INTO billing_transactions (provider_id, billing_usage_id, amount, currency, description, external_payment_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	err := r.pool.QueryRow(ctx, query,
		tx.ProviderID, tx.BillingUsageID, tx.Amount, tx.Currency, tx.Description, tx.ExternalPaymentID,
	).Scan(&tx.ID, &tx.CreatedAt)
	if err != nil {
		return fmt.Errorf("billing.repository: CreateTransaction: %w", err)
	}
	return nil
}

// ListTransactions returns the most recent transactions for a provider.
func (r *Repository) ListTransactions(ctx context.Context, providerID uuid.UUID, limit int) ([]BillingTransaction, error) {
	query := `
		SELECT id, provider_id, billing_usage_id, amount, currency, description, external_payment_id, created_at
		FROM billing_transactions
		WHERE provider_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, query, providerID, limit)
	if err != nil {
		return nil, fmt.Errorf("billing.repository: ListTransactions: %w", err)
	}
	defer rows.Close()

	var txns []BillingTransaction
	for rows.Next() {
		var tx BillingTransaction
		if err := rows.Scan(
			&tx.ID, &tx.ProviderID, &tx.BillingUsageID, &tx.Amount, &tx.Currency,
			&tx.Description, &tx.ExternalPaymentID, &tx.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("billing.repository: ListTransactions: %w", err)
		}
		txns = append(txns, tx)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("billing.repository: ListTransactions: %w", err)
	}
	return txns, nil
}

// MarkPaid marks a billing usage record as paid.
func (r *Repository) MarkPaid(ctx context.Context, usageID uuid.UUID) error {
	query := `UPDATE billing_usage SET is_paid = true, updated_at = now() WHERE id = $1`

	ct, err := r.pool.Exec(ctx, query, usageID)
	if err != nil {
		return fmt.Errorf("billing.repository: MarkPaid: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("billing.repository: MarkPaid: not found")
	}
	return nil
}

// CountUnpaidMonths counts months where is_paid=false and amount_due > 0 for a provider.
func (r *Repository) CountUnpaidMonths(ctx context.Context, providerID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM billing_usage
		WHERE provider_id = $1 AND is_paid = false AND amount_due > 0`

	var count int
	err := r.pool.QueryRow(ctx, query, providerID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("billing.repository: CountUnpaidMonths: %w", err)
	}
	return count, nil
}

// RecalculateUsage recalculates billable and amount_due from the current completed count.
func (r *Repository) RecalculateUsage(ctx context.Context, usageID uuid.UUID) error {
	query := `
		UPDATE billing_usage
		SET billable_appointments = GREATEST(0, completed_appointments - free_tier_limit),
			amount_due = LEAST($1, GREATEST(0, completed_appointments - free_tier_limit) * $2),
			updated_at = now()
		WHERE id = $3`

	ct, err := r.pool.Exec(ctx, query, MonthlyCap, PricePerAppointment, usageID)
	if err != nil {
		return fmt.Errorf("billing.repository: RecalculateUsage: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("billing.repository: RecalculateUsage: not found")
	}
	return nil
}

// ListAllProviderIDs returns all distinct provider IDs from billing_usage,
// plus any providers from the providers table (to ensure all providers get a usage row).
func (r *Repository) ListAllProviderIDs(ctx context.Context) ([]uuid.UUID, error) {
	query := `SELECT id FROM providers WHERE is_active = true`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("billing.repository: ListAllProviderIDs: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("billing.repository: ListAllProviderIDs: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("billing.repository: ListAllProviderIDs: %w", err)
	}
	return ids, nil
}

// CountCompletedForMonth counts completed appointments from the appointments table for a provider in a given month.
func (r *Repository) CountCompletedForMonth(ctx context.Context, providerID uuid.UUID, month time.Time) (int, error) {
	m := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	nextMonth := m.AddDate(0, 1, 0)

	query := `
		SELECT COUNT(*)
		FROM appointments
		WHERE provider_id = $1
		AND status = 'completed'
		AND date >= $2::date AND date < $3::date`

	var count int
	err := r.pool.QueryRow(ctx, query, providerID, m.Format("2006-01-02"), nextMonth.Format("2006-01-02")).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("billing.repository: CountCompletedForMonth: %w", err)
	}
	return count, nil
}

// SyncUsageTotals updates a billing_usage row with the actual completed count and recalculates.
func (r *Repository) SyncUsageTotals(ctx context.Context, usageID uuid.UUID, completed int) error {
	billable := CalculateBillable(completed, FreeTierLimit)
	amount := CalculateAmountDue(billable)
	// Round to 2 decimal places
	amount = math.Round(amount*100) / 100

	query := `
		UPDATE billing_usage
		SET completed_appointments = $1,
			billable_appointments = $2,
			amount_due = $3,
			updated_at = now()
		WHERE id = $4`

	_, err := r.pool.Exec(ctx, query, completed, billable, amount, usageID)
	if err != nil {
		return fmt.Errorf("billing.repository: SyncUsageTotals: %w", err)
	}
	return nil
}

// ListUnpaidWithAmountDue returns usage records with amount_due > 0 and is_paid = false.
func (r *Repository) ListUnpaidWithAmountDue(ctx context.Context) ([]BillingUsage, error) {
	query := `
		SELECT id, provider_id, month, completed_appointments, free_tier_limit,
			   billable_appointments, amount_due, is_paid, created_at, updated_at
		FROM billing_usage
		WHERE is_paid = false AND amount_due > 0
		ORDER BY month DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("billing.repository: ListUnpaidWithAmountDue: %w", err)
	}
	defer rows.Close()

	var usages []BillingUsage
	for rows.Next() {
		var u BillingUsage
		if err := rows.Scan(
			&u.ID, &u.ProviderID, &u.Month, &u.CompletedAppointments, &u.FreeTierLimit,
			&u.BillableAppointments, &u.AmountDue, &u.IsPaid, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("billing.repository: ListUnpaidWithAmountDue: %w", err)
		}
		usages = append(usages, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("billing.repository: ListUnpaidWithAmountDue: %w", err)
	}
	return usages, nil
}
