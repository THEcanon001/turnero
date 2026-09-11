package employee

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/THEcanon001/turnero/internal/platform/middleware"
)

const invitationCodeTTL = 48 * time.Hour

// Handler handles HTTP requests for employees.
type Handler struct {
	repo     *Repository
	validate *validator.Validate
}

// NewHandler creates a new employee handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{
		repo:     repo,
		validate: validator.New(),
	}
}

// List handles GET /v1/employees.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	providerID := middleware.GetProviderID(r.Context())

	employees, err := h.repo.ListByProvider(r.Context(), providerID)
	if err != nil {
		slog.Error("employee.handler: list: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list employees")
		return
	}

	if employees == nil {
		employees = []Employee{}
	}

	writeJSON(w, http.StatusOK, employees)
}

// GetByID handles GET /v1/employees/{id}.
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid employee ID")
		return
	}

	providerID := middleware.GetProviderID(r.Context())

	emp, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Employee not found")
		return
	}

	if emp.ProviderID != providerID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Not your employee")
		return
	}

	serviceIDs, err := h.repo.ListServiceIDs(r.Context(), emp.ID)
	if err != nil {
		slog.Error("employee.handler: list service ids: " + err.Error())
		serviceIDs = []uuid.UUID{}
	}

	resp := map[string]any{
		"employee":    emp,
		"service_ids": serviceIDs,
	}

	writeJSON(w, http.StatusOK, resp)
}

// Create handles POST /v1/employees.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateEmployeeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", formatValidationError(err))
		return
	}

	providerID := middleware.GetProviderID(r.Context())

	emp := &Employee{
		ProviderID: providerID,
		Name:       req.Name,
		Phone:      req.Phone,
		Role:       Role(req.Role),
		Email:      req.Email,
	}

	if err := h.repo.Create(r.Context(), emp); err != nil {
		slog.Error("employee.handler: create: " + err.Error())
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique") {
			writeError(w, http.StatusConflict, "CONFLICT", "Employee already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create employee")
		return
	}

	writeJSON(w, http.StatusCreated, emp)
}

// Update handles PUT /v1/employees/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid employee ID")
		return
	}

	var req UpdateEmployeeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", formatValidationError(err))
		return
	}

	providerID := middleware.GetProviderID(r.Context())

	emp, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Employee not found")
		return
	}

	if emp.ProviderID != providerID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Not your employee")
		return
	}

	emp.Name = req.Name
	emp.Phone = req.Phone
	if req.IsActive != nil {
		emp.IsActive = *req.IsActive
	}

	if err := h.repo.Update(r.Context(), emp); err != nil {
		slog.Error("employee.handler: update: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update employee")
		return
	}

	writeJSON(w, http.StatusOK, emp)
}

// Delete handles DELETE /v1/employees/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid employee ID")
		return
	}

	providerID := middleware.GetProviderID(r.Context())

	emp, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Employee not found")
		return
	}

	if emp.ProviderID != providerID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Not your employee")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		slog.Error("employee.handler: delete: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete employee")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateInvitation handles POST /v1/employees/invitations.
func (h *Handler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	var req CreateInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", formatValidationError(err))
		return
	}

	providerID := middleware.GetProviderID(r.Context())

	inv, err := h.repo.CreateInvitation(r.Context(), providerID, req.EmployeeName, invitationCodeTTL)
	if err != nil {
		slog.Error("employee.handler: create invitation: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create invitation")
		return
	}

	writeJSON(w, http.StatusCreated, inv)
}

// ListInvitations handles GET /v1/employees/invitations.
func (h *Handler) ListInvitations(w http.ResponseWriter, r *http.Request) {
	providerID := middleware.GetProviderID(r.Context())

	invitations, err := h.repo.ListInvitations(r.Context(), providerID)
	if err != nil {
		slog.Error("employee.handler: list invitations: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list invitations")
		return
	}

	if invitations == nil {
		invitations = []InvitationCode{}
	}

	writeJSON(w, http.StatusOK, invitations)
}

// AssignServices handles PUT /v1/employees/{id}/services.
func (h *Handler) AssignServices(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid employee ID")
		return
	}

	var req AssignServicesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", formatValidationError(err))
		return
	}

	providerID := middleware.GetProviderID(r.Context())

	emp, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Employee not found")
		return
	}

	if emp.ProviderID != providerID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Not your employee")
		return
	}

	if err := h.repo.AssignServices(r.Context(), id, req.ServiceIDs); err != nil {
		slog.Error("employee.handler: assign services: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to assign services")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"employee_id": id,
		"service_ids": req.ServiceIDs,
	})
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
