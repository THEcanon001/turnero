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

// InvitationCode represents a code that allows an employee to join a provider.
type InvitationCode struct {
	ID           uuid.UUID  `json:"id"`
	ProviderID   uuid.UUID  `json:"provider_id"`
	Code         string     `json:"code"`
	EmployeeName string     `json:"employee_name"`
	ExpiresAt    time.Time  `json:"expires_at"`
	UsedAt       *time.Time `json:"used_at,omitempty"`
	UsedBy       *uuid.UUID `json:"used_by,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// CreateEmployeeRequest is the input for creating an employee directly (admin).
type CreateEmployeeRequest struct {
	Name  string  `json:"name" validate:"required,min=2,max=100"`
	Phone string  `json:"phone" validate:"required"`
	Email *string `json:"email" validate:"omitempty,email"`
	Role  string  `json:"role" validate:"required,oneof=admin employee"`
}

// UpdateEmployeeRequest is the input for updating an employee.
type UpdateEmployeeRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=100"`
	Phone    string `json:"phone" validate:"required"`
	IsActive *bool  `json:"is_active"`
}

// CreateInvitationRequest is the input for generating an invitation code.
type CreateInvitationRequest struct {
	EmployeeName string `json:"employee_name" validate:"required,min=2,max=100"`
}

// JoinRequest is the input for an employee to join a provider via invitation code.
type JoinRequest struct {
	Code     string `json:"code" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Phone    string `json:"phone" validate:"required"`
}

// AssignServicesRequest is the input for assigning services to an employee.
type AssignServicesRequest struct {
	ServiceIDs []uuid.UUID `json:"service_ids" validate:"required,min=1"`
}
