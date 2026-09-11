package billing

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Service handles billing business logic.
type Service struct {
	repo *Repository
}

// NewService creates a new billing service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// RecordCompletion increments the completed appointment count for the current month.
// Returns the updated usage and whether the provider has exceeded the free tier.
func (s *Service) RecordCompletion(ctx context.Context, providerID uuid.UUID) (*BillingUsage, bool, error) {
	usage, err := s.repo.IncrementCompleted(ctx, providerID)
	if err != nil {
		return nil, false, fmt.Errorf("billing.service: RecordCompletion: %w", err)
	}

	exceededFreeTier := usage.CompletedAppointments > usage.FreeTierLimit
	return usage, exceededFreeTier, nil
}

// IsRestricted checks if a provider should be restricted.
// A provider is restricted if they have 3+ unpaid months with amount_due > 0.
func (s *Service) IsRestricted(ctx context.Context, providerID uuid.UUID) (bool, error) {
	count, err := s.repo.CountUnpaidMonths(ctx, providerID)
	if err != nil {
		return false, fmt.Errorf("billing.service: IsRestricted: %w", err)
	}
	return count >= GraceThreshold, nil
}

// GetCurrentUsage returns the current month's billing usage for a provider.
func (s *Service) GetCurrentUsage(ctx context.Context, providerID uuid.UUID) (*BillingUsage, error) {
	usage, err := s.repo.GetCurrentUsage(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("billing.service: GetCurrentUsage: %w", err)
	}
	return usage, nil
}
