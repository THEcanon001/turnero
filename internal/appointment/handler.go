package appointment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/THEcanon001/turnero/internal/employee"
	"github.com/THEcanon001/turnero/internal/platform/middleware"
	"github.com/THEcanon001/turnero/internal/provider"
)

const cancelDeadlineHours = 2

// BillingRecorder records completed appointments for billing purposes.
type BillingRecorder interface {
	RecordCompletion(ctx context.Context, providerID uuid.UUID) (exceededFreeTier bool, err error)
}

// Handler handles HTTP requests for appointments and availability.
type Handler struct {
	service         *Service
	repo            *Repository
	providerRepo    *provider.Repository
	employeeRepo    *employee.Repository
	validate        *validator.Validate
	billingRecorder BillingRecorder
}

// NewHandler creates a new appointment handler.
func NewHandler(service *Service, repo *Repository, providerRepo *provider.Repository, employeeRepo *employee.Repository) *Handler {
	return &Handler{
		service:      service,
		repo:         repo,
		providerRepo: providerRepo,
		employeeRepo: employeeRepo,
		validate:     validator.New(),
	}
}

// SetBillingRecorder sets the billing recorder for tracking completed appointments.
func (h *Handler) SetBillingRecorder(br BillingRecorder) {
	h.billingRecorder = br
}

// GetSlots handles GET /v1/providers/{slug}/employees/{employeeId}/slots?date=YYYY-MM-DD.
// Public endpoint — no auth required.
func (h *Handler) GetSlots(w http.ResponseWriter, r *http.Request) {
	employeeID, err := uuid.Parse(chi.URLParam(r, "employeeId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid employee ID")
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "date query param required (YYYY-MM-DD)")
		return
	}

	slots, err := h.service.GetAvailableSlots(r.Context(), employeeID, date)
	if err != nil {
		slog.Error("appointment.handler: get slots: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get slots")
		return
	}

	writeJSON(w, http.StatusOK, slots)
}

// GetProviderProfile handles GET /v1/providers/{slug}.
// Public endpoint — returns provider info + services.
func (h *Handler) GetProviderProfile(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "slug is required")
		return
	}

	p, err := h.providerRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Provider not found")
		return
	}

	// Get services
	services, err := h.providerRepo.ListServices(r.Context(), p.ID)
	if err != nil {
		slog.Error("appointment.handler: list services: " + err.Error())
		services = []provider.Service{}
	}

	resp := map[string]any{
		"id":            p.ID,
		"name":          p.Name,
		"slug":          p.Slug,
		"phone":         p.Phone,
		"address":       p.Address,
		"type":          p.Type,
		"services":      services,
		"whatsapp_link": WhatsAppLink(p.Phone, ""),
	}

	writeJSON(w, http.StatusOK, resp)
}

// Create handles POST /v1/appointments.
// Public endpoint — clients book without authentication.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", formatValidationError(err))
		return
	}

	// Resolve provider by slug
	p, err := h.providerRepo.GetBySlug(r.Context(), req.ProviderSlug)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Provider not found")
		return
	}

	// Resolve employee ID: if nil, find "any available"
	var employeeID uuid.UUID
	var endTime string

	if req.EmployeeID != nil && *req.EmployeeID != uuid.Nil {
		employeeID = *req.EmployeeID

		slots, err := h.service.GetAvailableSlots(r.Context(), employeeID, req.Date)
		if err != nil {
			slog.Error("appointment.handler: check availability: " + err.Error())
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check availability")
			return
		}

		var slotFound bool
		for _, slot := range slots {
			if slot.StartTime == req.StartTime && slot.Available {
				slotFound = true
				endTime = slot.EndTime
				break
			}
		}

		if !slotFound {
			writeError(w, http.StatusConflict, "SLOT_TAKEN", "This time slot is not available")
			return
		}
	} else {
		// "Any available" — find first employee with the requested slot open
		employees, err := h.employeeRepo.ListByProvider(r.Context(), p.ID)
		if err != nil {
			slog.Error("appointment.handler: list employees: " + err.Error())
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to find employees")
			return
		}

		var found bool
		for _, emp := range employees {
			if !emp.IsActive {
				continue
			}
			slots, err := h.service.GetAvailableSlots(r.Context(), emp.ID, req.Date)
			if err != nil {
				continue
			}
			for _, slot := range slots {
				if slot.StartTime == req.StartTime && slot.Available {
					employeeID = emp.ID
					endTime = slot.EndTime
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		if !found {
			writeError(w, http.StatusConflict, "SLOT_TAKEN", "No employees available at this time")
			return
		}
	}

	apt := &Appointment{
		ProviderID:  p.ID,
		EmployeeID:  employeeID,
		ServiceID:   req.ServiceID,
		ClientName:  req.ClientName,
		ClientPhone: req.ClientPhone,
		Date:        req.Date,
		StartTime:   req.StartTime,
		EndTime:     endTime,
		Notes:       req.Notes,
	}

	if err := h.repo.Create(r.Context(), apt); err != nil {
		if strings.Contains(err.Error(), "slot already taken") {
			writeError(w, http.StatusConflict, "SLOT_TAKEN", "This time slot was just booked")
			return
		}
		slog.Error("appointment.handler: create: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create appointment")
		return
	}

	resp := map[string]any{
		"appointment":   apt,
		"whatsapp_link": AppointmentWhatsAppLink(p.Phone, apt.ClientName, apt.Date, apt.StartTime),
	}

	writeJSON(w, http.StatusCreated, resp)
}

// GetByID handles GET /v1/appointments/{id}.
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid appointment ID")
		return
	}

	apt, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Appointment not found")
		return
	}

	p, err := h.providerRepo.GetByID(r.Context(), apt.ProviderID)
	if err != nil {
		writeJSON(w, http.StatusOK, apt)
		return
	}

	resp := map[string]any{
		"appointment":   apt,
		"whatsapp_link": AppointmentWhatsAppLink(p.Phone, apt.ClientName, apt.Date, apt.StartTime),
	}

	writeJSON(w, http.StatusOK, resp)
}

// Cancel handles POST /v1/appointments/{id}/cancel.
// Public endpoint — clients can cancel if more than 2 hours before the appointment.
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid appointment ID")
		return
	}

	var req CancelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Empty body is OK for cancel
		req = CancelRequest{}
	}

	apt, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Appointment not found")
		return
	}

	if apt.Status != StatusConfirmed {
		writeError(w, http.StatusUnprocessableEntity, "BUSINESS_RULE_VIOLATION",
			fmt.Sprintf("Cannot cancel appointment with status '%s'", apt.Status))
		return
	}

	// Time restriction: client can't cancel within 2 hours of appointment
	if err := checkCancelDeadline(apt.Date, apt.StartTime); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "CANCEL_TOO_LATE", err.Error())
		return
	}

	if err := h.repo.UpdateStatus(r.Context(), id, StatusCancelled, req.Reason); err != nil {
		slog.Error("appointment.handler: cancel: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to cancel")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// ProviderCancel handles POST /v1/appointments/{id}/provider-cancel.
// Authenticated endpoint — providers can cancel any time without restriction.
func (h *Handler) ProviderCancel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid appointment ID")
		return
	}

	var req CancelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = CancelRequest{}
	}

	apt, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Appointment not found")
		return
	}

	providerID := middleware.GetProviderID(r.Context())
	if apt.ProviderID != providerID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Not your appointment")
		return
	}

	if apt.Status != StatusConfirmed {
		writeError(w, http.StatusUnprocessableEntity, "BUSINESS_RULE_VIOLATION",
			fmt.Sprintf("Cannot cancel appointment with status '%s'", apt.Status))
		return
	}

	if err := h.repo.UpdateStatus(r.Context(), id, StatusCancelled, req.Reason); err != nil {
		slog.Error("appointment.handler: provider cancel: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to cancel")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// Reschedule handles POST /v1/appointments/{id}/reschedule.
// Public endpoint — cancels existing and creates new in one operation.
func (h *Handler) Reschedule(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid appointment ID")
		return
	}

	var req RescheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", formatValidationError(err))
		return
	}

	apt, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Appointment not found")
		return
	}

	if apt.Status != StatusConfirmed {
		writeError(w, http.StatusUnprocessableEntity, "BUSINESS_RULE_VIOLATION",
			fmt.Sprintf("Cannot reschedule appointment with status '%s'", apt.Status))
		return
	}

	if err := checkCancelDeadline(apt.Date, apt.StartTime); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "CANCEL_TOO_LATE", err.Error())
		return
	}

	// Verify new slot is available
	slots, err := h.service.GetAvailableSlots(r.Context(), apt.EmployeeID, req.Date)
	if err != nil {
		slog.Error("appointment.handler: reschedule check slots: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check availability")
		return
	}

	var endTime string
	var slotFound bool
	for _, slot := range slots {
		if slot.StartTime == req.StartTime && slot.Available {
			slotFound = true
			endTime = slot.EndTime
			break
		}
	}

	if !slotFound {
		writeError(w, http.StatusConflict, "SLOT_TAKEN", "New time slot is not available")
		return
	}

	// Cancel old appointment
	reason := "rescheduled"
	if err := h.repo.UpdateStatus(r.Context(), id, StatusCancelled, &reason); err != nil {
		slog.Error("appointment.handler: reschedule cancel: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to reschedule")
		return
	}

	// Create new appointment
	newApt := &Appointment{
		ProviderID:  apt.ProviderID,
		EmployeeID:  apt.EmployeeID,
		ServiceID:   apt.ServiceID,
		ClientName:  apt.ClientName,
		ClientPhone: apt.ClientPhone,
		Date:        req.Date,
		StartTime:   req.StartTime,
		EndTime:     endTime,
		Notes:       apt.Notes,
	}

	if err := h.repo.Create(r.Context(), newApt); err != nil {
		slog.Error("appointment.handler: reschedule create: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create rescheduled appointment")
		return
	}

	writeJSON(w, http.StatusCreated, newApt)
}

// Reassign handles PUT /v1/appointments/{id}/reassign.
// Authenticated endpoint — admin moves appointment to another employee.
func (h *Handler) Reassign(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid appointment ID")
		return
	}

	var req ReassignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", formatValidationError(err))
		return
	}

	apt, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Appointment not found")
		return
	}

	providerID := middleware.GetProviderID(r.Context())
	if apt.ProviderID != providerID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Not your appointment")
		return
	}

	if apt.Status != StatusConfirmed {
		writeError(w, http.StatusUnprocessableEntity, "BUSINESS_RULE_VIOLATION",
			fmt.Sprintf("Cannot reassign appointment with status '%s'", apt.Status))
		return
	}

	// Verify new employee belongs to same provider
	newEmp, err := h.employeeRepo.GetByID(r.Context(), req.EmployeeID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Employee not found")
		return
	}
	if newEmp.ProviderID != providerID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Employee does not belong to your provider")
		return
	}

	// Verify the new employee has the slot available
	slots, err := h.service.GetAvailableSlots(r.Context(), req.EmployeeID, apt.Date)
	if err != nil {
		slog.Error("appointment.handler: reassign check slots: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check availability")
		return
	}

	var slotAvailable bool
	for _, slot := range slots {
		if slot.StartTime == apt.StartTime && slot.Available {
			slotAvailable = true
			break
		}
	}

	if !slotAvailable {
		writeError(w, http.StatusConflict, "SLOT_TAKEN", "Employee is not available at this time")
		return
	}

	if err := h.repo.UpdateEmployee(r.Context(), id, req.EmployeeID); err != nil {
		slog.Error("appointment.handler: reassign: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to reassign")
		return
	}

	apt.EmployeeID = req.EmployeeID
	writeJSON(w, http.StatusOK, apt)
}

// WalkIn handles POST /v1/appointments/walk-in.
// Authenticated endpoint — providers create manual appointments for walk-in clients.
func (h *Handler) WalkIn(w http.ResponseWriter, r *http.Request) {
	var req WalkInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", formatValidationError(err))
		return
	}

	providerID := middleware.GetProviderID(r.Context())

	// Verify employee belongs to provider
	emp, err := h.employeeRepo.GetByID(r.Context(), req.EmployeeID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Employee not found")
		return
	}
	if emp.ProviderID != providerID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Employee does not belong to your provider")
		return
	}

	// Check slot availability
	slots, err := h.service.GetAvailableSlots(r.Context(), req.EmployeeID, req.Date)
	if err != nil {
		slog.Error("appointment.handler: walk-in check slots: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check availability")
		return
	}

	var endTime string
	var slotFound bool
	for _, slot := range slots {
		if slot.StartTime == req.StartTime && slot.Available {
			slotFound = true
			endTime = slot.EndTime
			break
		}
	}

	if !slotFound {
		writeError(w, http.StatusConflict, "SLOT_TAKEN", "This time slot is not available")
		return
	}

	apt := &Appointment{
		ProviderID:  providerID,
		EmployeeID:  req.EmployeeID,
		ServiceID:   req.ServiceID,
		ClientName:  req.ClientName,
		ClientPhone: req.ClientPhone,
		Date:        req.Date,
		StartTime:   req.StartTime,
		EndTime:     endTime,
		Notes:       req.Notes,
	}

	if err := h.repo.Create(r.Context(), apt); err != nil {
		if strings.Contains(err.Error(), "slot already taken") {
			writeError(w, http.StatusConflict, "SLOT_TAKEN", "This time slot was just booked")
			return
		}
		slog.Error("appointment.handler: walk-in create: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create appointment")
		return
	}

	writeJSON(w, http.StatusCreated, apt)
}

// checkCancelDeadline verifies the appointment isn't within cancelDeadlineHours.
func checkCancelDeadline(date, startTime string) error {
	appointmentTime, err := time.Parse("2006-01-02 15:04", date+" "+startTime)
	if err != nil {
		return nil // if we can't parse, allow cancellation
	}

	deadline := appointmentTime.Add(-time.Duration(cancelDeadlineHours) * time.Hour)
	if time.Now().After(deadline) {
		return fmt.Errorf("cannot cancel within %d hours of appointment time", cancelDeadlineHours)
	}
	return nil
}

// UpdateStatus handles PUT /v1/appointments/{id}/status.
// Authenticated endpoint — for providers to mark complete/no-show.
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid appointment ID")
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", formatValidationError(err))
		return
	}

	// Verify the appointment belongs to the authenticated provider
	apt, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Appointment not found")
		return
	}

	providerID := middleware.GetProviderID(r.Context())
	if apt.ProviderID != providerID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Not your appointment")
		return
	}

	if err := h.repo.UpdateStatus(r.Context(), id, req.Status, nil); err != nil {
		slog.Error("appointment.handler: update status: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update status")
		return
	}

	// Record billing when an appointment is completed
	if req.Status == StatusCompleted && h.billingRecorder != nil {
		exceeded, err := h.billingRecorder.RecordCompletion(r.Context(), providerID)
		if err != nil {
			slog.Error("appointment.handler: record billing: " + err.Error())
		} else if exceeded {
			slog.Info("appointment.handler: provider exceeded free tier",
				slog.String("provider_id", providerID.String()))
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": string(req.Status)})
}

// ListByProvider handles GET /v1/appointments?date=YYYY-MM-DD&status=confirmed&page=1&per_page=20.
// Authenticated endpoint.
func (h *Handler) ListByProvider(w http.ResponseWriter, r *http.Request) {
	providerID := middleware.GetProviderID(r.Context())

	filter := ListFilter{
		ProviderID: &providerID,
		Date:       r.URL.Query().Get("date"),
		Page:       1,
		PerPage:    20,
	}

	if eid := r.URL.Query().Get("employee_id"); eid != "" {
		empID, err := uuid.Parse(eid)
		if err == nil {
			filter.EmployeeID = &empID
		}
	}

	if s := r.URL.Query().Get("status"); s != "" {
		status := Status(s)
		filter.Status = &status
	}

	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			filter.Page = v
		}
	}
	if pp := r.URL.Query().Get("per_page"); pp != "" {
		if v, err := strconv.Atoi(pp); err == nil {
			filter.PerPage = v
		}
	}

	appointments, total, err := h.repo.List(r.Context(), filter)
	if err != nil {
		slog.Error("appointment.handler: list: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list appointments")
		return
	}

	if appointments == nil {
		appointments = []Appointment{}
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	writeJSON(w, http.StatusOK, appointments)
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
