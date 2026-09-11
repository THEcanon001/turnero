package schedule

import (
	"time"

	"github.com/google/uuid"
)

// Schedule represents a weekly schedule slot for an employee.
type Schedule struct {
	ID                    uuid.UUID `json:"id"`
	EmployeeID            uuid.UUID `json:"employee_id"`
	DayOfWeek             int16     `json:"day_of_week"` // 0=Sunday, 6=Saturday
	StartTime             string    `json:"start_time"`  // HH:MM format
	EndTime               string    `json:"end_time"`    // HH:MM format
	SlotDurationMinutes   int16     `json:"slot_duration_minutes"`
	BreakAfterSlotMinutes int16     `json:"break_after_slot_minutes"`
	IsActive              bool      `json:"is_active"`
	CreatedAt             time.Time `json:"created_at"`
}

// CreateScheduleRequest is the input for setting a schedule for a day.
type CreateScheduleRequest struct {
	DayOfWeek             int16  `json:"day_of_week" validate:"min=0,max=6"`
	StartTime             string `json:"start_time" validate:"required"`
	EndTime               string `json:"end_time" validate:"required"`
	SlotDurationMinutes   int16  `json:"slot_duration_minutes" validate:"required,min=5,max=480"`
	BreakAfterSlotMinutes int16  `json:"break_after_slot_minutes" validate:"min=0,max=120"`
}

// ScheduleException represents a day override (day off or special hours).
type ScheduleException struct {
	ID          uuid.UUID `json:"id"`
	EmployeeID  uuid.UUID `json:"employee_id"`
	Date        string    `json:"date"` // YYYY-MM-DD
	IsAvailable bool      `json:"is_available"`
	StartTime   *string   `json:"start_time,omitempty"`
	EndTime     *string   `json:"end_time,omitempty"`
	Reason      *string   `json:"reason,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateExceptionRequest is the input for adding a schedule exception.
type CreateExceptionRequest struct {
	Date        string  `json:"date" validate:"required"`
	IsAvailable bool    `json:"is_available"`
	StartTime   *string `json:"start_time"`
	EndTime     *string `json:"end_time"`
	Reason      *string `json:"reason" validate:"omitempty,max=200"`
}
