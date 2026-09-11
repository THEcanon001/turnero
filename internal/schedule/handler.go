package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/THEcanon001/turnero/internal/employee"
	"github.com/THEcanon001/turnero/internal/platform/middleware"
)

// AppointmentCanceller is the interface needed to batch-cancel appointments.
type AppointmentCanceller interface {
	CancelByEmployeeAndDateRange(ctx context.Context, employeeID uuid.UUID, fromDate, toDate, reason string) (int64, error)
}

// Handler handles HTTP requests for schedules.
type Handler struct {
	scheduleRepo    *Repository
	employeeRepo    *employee.Repository
	aptCanceller    AppointmentCanceller
	validate        *validator.Validate
}

// NewHandler creates a new schedule handler.
func NewHandler(scheduleRepo *Repository, employeeRepo *employee.Repository) *Handler {
	return &Handler{
		scheduleRepo: scheduleRepo,
		employeeRepo: employeeRepo,
		validate:     validator.New(),
	}
}

// SetAppointmentCanceller sets the appointment canceller for batch operations.
func (h *Handler) SetAppointmentCanceller(c AppointmentCanceller) {
	h.aptCanceller = c
}

// SetSchedule handles PUT /v1/employees/{employeeId}/schedules.
// Upserts schedules for a given employee (bulk set weekly schedule).
func (h *Handler) SetSchedule(w http.ResponseWriter, r *http.Request) {
	employeeID, err := uuid.Parse(chi.URLParam(r, "employeeId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid employee ID")
		return
	}

	if err := h.verifyEmployeeOwnership(r, employeeID); err != nil {
		writeError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
		return
	}

	var reqs []CreateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	for _, req := range reqs {
		if err := h.validate.Struct(req); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", formatValidationError(err))
			return
		}
		if err := ValidateTimeRange(req.StartTime, req.EndTime); err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
	}

	var schedules []Schedule
	for _, req := range reqs {
		s := &Schedule{
			EmployeeID:            employeeID,
			DayOfWeek:             req.DayOfWeek,
			StartTime:             req.StartTime,
			EndTime:               req.EndTime,
			SlotDurationMinutes:   req.SlotDurationMinutes,
			BreakAfterSlotMinutes: req.BreakAfterSlotMinutes,
		}
		if err := h.scheduleRepo.Upsert(r.Context(), s); err != nil {
			slog.Error("schedule.handler: set schedule: " + err.Error())
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to save schedule")
			return
		}
		schedules = append(schedules, *s)
	}

	writeJSON(w, http.StatusOK, schedules)
}

// GetSchedule handles GET /v1/employees/{employeeId}/schedules.
func (h *Handler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	employeeID, err := uuid.Parse(chi.URLParam(r, "employeeId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid employee ID")
		return
	}

	schedules, err := h.scheduleRepo.ListByEmployee(r.Context(), employeeID)
	if err != nil {
		slog.Error("schedule.handler: get schedule: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get schedule")
		return
	}

	if schedules == nil {
		schedules = []Schedule{}
	}
	writeJSON(w, http.StatusOK, schedules)
}

// AddException handles POST /v1/employees/{employeeId}/schedule-exceptions.
func (h *Handler) AddException(w http.ResponseWriter, r *http.Request) {
	employeeID, err := uuid.Parse(chi.URLParam(r, "employeeId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid employee ID")
		return
	}

	if err := h.verifyEmployeeOwnership(r, employeeID); err != nil {
		writeError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
		return
	}

	var req CreateExceptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", formatValidationError(err))
		return
	}

	exc := &ScheduleException{
		EmployeeID:  employeeID,
		Date:        req.Date,
		IsAvailable: req.IsAvailable,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		Reason:      req.Reason,
	}

	if err := h.scheduleRepo.CreateException(r.Context(), exc); err != nil {
		slog.Error("schedule.handler: add exception: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to add exception")
		return
	}

	// If marking day as unavailable, batch-cancel affected appointments
	var cancelledCount int64
	if !exc.IsAvailable && h.aptCanceller != nil {
		reason := "day blocked by provider"
		if exc.Reason != nil && *exc.Reason != "" {
			reason = *exc.Reason
		}
		n, err := h.aptCanceller.CancelByEmployeeAndDateRange(r.Context(), employeeID, exc.Date, exc.Date, reason)
		if err != nil {
			slog.Error("schedule.handler: batch cancel: " + err.Error())
		}
		cancelledCount = n
	}

	resp := map[string]any{
		"exception":               exc,
		"cancelled_appointments": cancelledCount,
	}
	writeJSON(w, http.StatusCreated, resp)
}

// ListExceptions handles GET /v1/employees/{employeeId}/schedule-exceptions.
func (h *Handler) ListExceptions(w http.ResponseWriter, r *http.Request) {
	employeeID, err := uuid.Parse(chi.URLParam(r, "employeeId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid employee ID")
		return
	}

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "from and to query params required (YYYY-MM-DD)")
		return
	}

	exceptions, err := h.scheduleRepo.ListExceptions(r.Context(), employeeID, from, to)
	if err != nil {
		slog.Error("schedule.handler: list exceptions: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list exceptions")
		return
	}

	if exceptions == nil {
		exceptions = []ScheduleException{}
	}
	writeJSON(w, http.StatusOK, exceptions)
}

// DeleteException handles DELETE /v1/schedule-exceptions/{id}.
func (h *Handler) DeleteException(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid exception ID")
		return
	}

	// Verify ownership via the exception's employee
	exc, err := h.scheduleRepo.GetExceptionByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Exception not found")
		return
	}

	if err := h.verifyEmployeeOwnership(r, exc.EmployeeID); err != nil {
		writeError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
		return
	}

	if err := h.scheduleRepo.DeleteException(r.Context(), id); err != nil {
		slog.Error("schedule.handler: delete exception: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Delete failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// verifyEmployeeOwnership checks that the employee belongs to the authenticated provider.
func (h *Handler) verifyEmployeeOwnership(r *http.Request, employeeID uuid.UUID) error {
	providerID := middleware.GetProviderID(r.Context())

	emp, err := h.employeeRepo.GetByID(r.Context(), employeeID)
	if err != nil {
		return err
	}

	if emp.ProviderID != providerID {
		return fmt.Errorf("employee does not belong to your provider")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
		"code":  code,
	})
}

// ValidateTimeRange ensures start_time is before end_time.
func ValidateTimeRange(startTime, endTime string) error {
	start, err := time.Parse("15:04", startTime)
	if err != nil {
		return fmt.Errorf("invalid start_time format, expected HH:MM")
	}
	end, err := time.Parse("15:04", endTime)
	if err != nil {
		return fmt.Errorf("invalid end_time format, expected HH:MM")
	}
	if !start.Before(end) {
		return fmt.Errorf("start_time must be before end_time")
	}
	return nil
}

func formatValidationError(err error) string {
	if ve, ok := err.(validator.ValidationErrors); ok {
		var msgs []string
		for _, fe := range ve {
			msgs = append(msgs, fe.Field()+" failed on "+fe.Tag())
		}
		return strings.Join(msgs, "; ")
	}
	return err.Error()
}
