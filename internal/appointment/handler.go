package appointment

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/THEcanon001/turnero/internal/platform/middleware"
	"github.com/THEcanon001/turnero/internal/provider"
)

// Handler handles HTTP requests for appointments and availability.
type Handler struct {
	service      *Service
	repo         *Repository
	providerRepo *provider.Repository
	validate     *validator.Validate
}

// NewHandler creates a new appointment handler.
func NewHandler(service *Service, repo *Repository, providerRepo *provider.Repository) *Handler {
	return &Handler{
		service:      service,
		repo:         repo,
		providerRepo: providerRepo,
		validate:     validator.New(),
	}
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
		"id":       p.ID,
		"name":     p.Name,
		"slug":     p.Slug,
		"phone":    p.Phone,
		"address":  p.Address,
		"type":     p.Type,
		"services": services,
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

	// Check slot availability
	slots, err := h.service.GetAvailableSlots(r.Context(), req.EmployeeID, req.Date)
	if err != nil {
		slog.Error("appointment.handler: check availability: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check availability")
		return
	}

	var slotFound bool
	var endTime string
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
		ProviderID:  p.ID,
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
		slog.Error("appointment.handler: create: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create appointment")
		return
	}

	writeJSON(w, http.StatusCreated, apt)
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

	writeJSON(w, http.StatusOK, apt)
}

// Cancel handles POST /v1/appointments/{id}/cancel.
// Public endpoint — verified by client phone.
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

	if err := h.repo.UpdateStatus(r.Context(), id, StatusCancelled, req.Reason); err != nil {
		slog.Error("appointment.handler: cancel: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to cancel")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
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
