package employee

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleEmployee Role = "employee"
)

// Employee represents a worker belonging to a provider.
type Employee struct {
	ID           uuid.UUID `json:"id"`
	ProviderID   uuid.UUID `json:"provider_id"`
	Name         string    `json:"name"`
	Phone        string    `json:"phone"`
	Role         Role      `json:"role"`
	Email        *string   `json:"email,omitempty"`
	PasswordHash *string   `json:"-"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
