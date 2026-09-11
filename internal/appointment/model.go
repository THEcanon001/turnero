package appointment

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusConfirmed Status = "confirmed"
	StatusCancelled Status = "cancelled"
	StatusCompleted Status = "completed"
	StatusNoShow    Status = "no_show"
)

// Appointment represents a booked time slot.
type Appointment struct {
	ID                 uuid.UUID  `json:"id"`
	ProviderID         uuid.UUID  `json:"provider_id"`
	EmployeeID         uuid.UUID  `json:"employee_id"`
	ServiceID          *uuid.UUID `json:"service_id,omitempty"`
	ClientName         string     `json:"client_name"`
	ClientPhone        string     `json:"client_phone"`
	Date               string     `json:"date"`       // YYYY-MM-DD
	StartTime          string     `json:"start_time"`  // HH:MM
	EndTime            string     `json:"end_time"`    // HH:MM
	Status             Status     `json:"status"`
	Notes              *string    `json:"notes,omitempty"`
	CancellationReason *string    `json:"cancellation_reason,omitempty"`
	ReminderSent       bool       `json:"reminder_sent"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// CreateRequest is the input for booking an appointment.
// EmployeeID can be omitted or set to nil UUID for "any available" assignment.
type CreateRequest struct {
	ProviderSlug string     `json:"provider_slug" validate:"required"`
	EmployeeID   *uuid.UUID `json:"employee_id"`
	ServiceID    *uuid.UUID `json:"service_id"`
	ClientName   string     `json:"client_name" validate:"required,min=2,max=100"`
	ClientPhone  string     `json:"client_phone" validate:"required"`
	Date         string     `json:"date" validate:"required"`
	StartTime    string     `json:"start_time" validate:"required"`
	Notes        *string    `json:"notes" validate:"omitempty,max=500"`
}

// CancelRequest is the input for cancelling an appointment.
type CancelRequest struct {
	Reason *string `json:"reason" validate:"omitempty,max=500"`
}

// UpdateStatusRequest is used by providers to change appointment status.
type UpdateStatusRequest struct {
	Status Status `json:"status" validate:"required,oneof=confirmed cancelled completed no_show"`
}

// Slot represents an available time slot.
type Slot struct {
	StartTime string `json:"start_time"` // HH:MM
	EndTime   string `json:"end_time"`   // HH:MM
	Available bool   `json:"available"`
}

// ListFilter holds filters for listing appointments.
type ListFilter struct {
	ProviderID *uuid.UUID
	EmployeeID *uuid.UUID
	Date       string
	Status     *Status
	Page       int
	PerPage    int
}
