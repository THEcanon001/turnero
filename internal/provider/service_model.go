package provider

import (
	"time"

	"github.com/google/uuid"
)

// Service represents a business service offered by a provider (e.g., "Haircut", "Massage").
type Service struct {
	ID              uuid.UUID `json:"id"`
	ProviderID      uuid.UUID `json:"provider_id"`
	Name            string    `json:"name"`
	Description     *string   `json:"description,omitempty"`
	DurationMinutes int16     `json:"duration_minutes"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateServiceRequest is the input for creating a service.
type CreateServiceRequest struct {
	Name            string  `json:"name" validate:"required,min=2,max=100"`
	Description     *string `json:"description" validate:"omitempty,max=500"`
	DurationMinutes int16   `json:"duration_minutes" validate:"required,min=5,max=480"`
}

// UpdateServiceRequest is the input for updating a service.
type UpdateServiceRequest struct {
	Name            string  `json:"name" validate:"required,min=2,max=100"`
	Description     *string `json:"description" validate:"omitempty,max=500"`
	DurationMinutes int16   `json:"duration_minutes" validate:"required,min=5,max=480"`
	IsActive        bool    `json:"is_active"`
}
