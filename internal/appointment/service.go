package appointment

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/THEcanon001/turnero/internal/schedule"
)

// Service handles appointment business logic including availability calculation.
type Service struct {
	appointmentRepo *Repository
	scheduleRepo    *schedule.Repository
}

// NewService creates a new appointment service.
func NewService(appointmentRepo *Repository, scheduleRepo *schedule.Repository) *Service {
	return &Service{
		appointmentRepo: appointmentRepo,
		scheduleRepo:    scheduleRepo,
	}
}

// GetAvailableSlots calculates available slots for an employee on a given date.
func (s *Service) GetAvailableSlots(ctx context.Context, employeeID uuid.UUID, date string) ([]Slot, error) {
	// Parse the date to get the day of week
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("appointment.service: invalid date format: %w", err)
	}

	dayOfWeek := int16(d.Weekday())

	// Check for exceptions on this date
	exceptions, err := s.scheduleRepo.ListExceptions(ctx, employeeID, date, date)
	if err != nil {
		return nil, fmt.Errorf("appointment.service: %w", err)
	}

	// If there's an exception marking the day as unavailable, return empty
	for _, exc := range exceptions {
		if !exc.IsAvailable {
			return []Slot{}, nil
		}
	}

	// Get the schedule for this day of week
	schedules, err := s.scheduleRepo.ListByEmployee(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("appointment.service: %w", err)
	}

	var daySchedule *schedule.Schedule
	for i := range schedules {
		if schedules[i].DayOfWeek == dayOfWeek && schedules[i].IsActive {
			daySchedule = &schedules[i]
			break
		}
	}

	if daySchedule == nil {
		return []Slot{}, nil
	}

	// Determine working hours (exception overrides schedule)
	startTimeStr := daySchedule.StartTime
	endTimeStr := daySchedule.EndTime
	for _, exc := range exceptions {
		if exc.IsAvailable && exc.StartTime != nil && exc.EndTime != nil {
			startTimeStr = *exc.StartTime
			endTimeStr = *exc.EndTime
		}
	}

	startTime, err := parseTime(startTimeStr)
	if err != nil {
		return nil, fmt.Errorf("appointment.service: parse start time: %w", err)
	}
	endTime, err := parseTime(endTimeStr)
	if err != nil {
		return nil, fmt.Errorf("appointment.service: parse end time: %w", err)
	}

	// Get existing confirmed appointments for this day
	existing, err := s.appointmentRepo.ListByEmployeeAndDate(ctx, employeeID, date)
	if err != nil {
		return nil, fmt.Errorf("appointment.service: %w", err)
	}

	// Build set of booked start times for fast lookup
	booked := make(map[string]bool)
	for _, a := range existing {
		booked[a.StartTime] = true
	}

	// Generate slots
	slotDuration := time.Duration(daySchedule.SlotDurationMinutes) * time.Minute
	breakDuration := time.Duration(daySchedule.BreakAfterSlotMinutes) * time.Minute

	var slots []Slot
	current := startTime
	for current.Add(slotDuration).Before(endTime) || current.Add(slotDuration).Equal(endTime) {
		slotStart := current.Format("15:04")
		slotEnd := current.Add(slotDuration).Format("15:04")

		slots = append(slots, Slot{
			StartTime: slotStart,
			EndTime:   slotEnd,
			Available: !booked[slotStart],
		})

		current = current.Add(slotDuration + breakDuration)
	}

	return slots, nil
}

// parseTime parses a time string in HH:MM or HH:MM:SS format.
func parseTime(s string) (time.Time, error) {
	// Try HH:MM:SS first (what PostgreSQL returns)
	t, err := time.Parse("15:04:05", s)
	if err == nil {
		return t, nil
	}
	// Try HH:MM
	return time.Parse("15:04", s)
}
