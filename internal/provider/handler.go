package provider

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/THEcanon001/turnero/internal/platform/middleware"
)

// Handler handles HTTP requests for provider and services.
type Handler struct {
	repo     *Repository
	validate *validator.Validate
}

// NewHandler creates a new provider handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{
		repo:     repo,
		validate: validator.New(),
	}
}

// GetMe handles GET /v1/provider/me.
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	providerID := middleware.GetProviderID(r.Context())

	p, err := h.repo.GetByID(r.Context(), providerID)
	if err != nil {
		slog.Error("provider.handler: get me: " + err.Error())
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Provider not found")
		return
	}

	writeJSON(w, http.StatusOK, p)
}

// UpdateMe handles PUT /v1/provider/me.
func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	providerID := middleware.GetProviderID(r.Context())

	var req struct {
		Name         string  `json:"name" validate:"required,min=2,max=100"`
		Phone        string  `json:"phone" validate:"required"`
		Address      *string `json:"address"`
		Timezone     string  `json:"timezone" validate:"required"`
		BusinessName *string `json:"business_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", formatValidationError(err))
		return
	}

	p, err := h.repo.GetByID(r.Context(), providerID)
	if err != nil {
		slog.Error("provider.handler: update me: " + err.Error())
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Provider not found")
		return
	}

	p.Name = req.Name
	p.Phone = req.Phone
	p.Address = req.Address
	p.Timezone = req.Timezone
	if req.BusinessName != nil {
		p.BusinessName = req.BusinessName
	}

	if err := h.repo.Update(r.Context(), p); err != nil {
		slog.Error("provider.handler: update me: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Update failed")
		return
	}

	writeJSON(w, http.StatusOK, p)
}

// CreateService handles POST /v1/services.
func (h *Handler) CreateService(w http.ResponseWriter, r *http.Request) {
	providerID := middleware.GetProviderID(r.Context())

	var req CreateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", formatValidationError(err))
		return
	}

	svc := &Service{
		ProviderID:      providerID,
		Name:            req.Name,
		Description:     req.Description,
		DurationMinutes: req.DurationMinutes,
	}

	if err := h.repo.CreateService(r.Context(), svc); err != nil {
		slog.Error("provider.handler: create service: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create service")
		return
	}

	writeJSON(w, http.StatusCreated, svc)
}

// ListServices handles GET /v1/services.
func (h *Handler) ListServices(w http.ResponseWriter, r *http.Request) {
	providerID := middleware.GetProviderID(r.Context())

	services, err := h.repo.ListServices(r.Context(), providerID)
	if err != nil {
		slog.Error("provider.handler: list services: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list services")
		return
	}

	if services == nil {
		services = []Service{}
	}
	writeJSON(w, http.StatusOK, services)
}

// GetService handles GET /v1/services/{id}.
func (h *Handler) GetService(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid service ID")
		return
	}

	svc, err := h.repo.GetServiceByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Service not found")
		return
	}

	// Verify ownership
	providerID := middleware.GetProviderID(r.Context())
	if svc.ProviderID != providerID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Not your service")
		return
	}

	writeJSON(w, http.StatusOK, svc)
}

// UpdateService handles PUT /v1/services/{id}.
func (h *Handler) UpdateService(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid service ID")
		return
	}

	var req UpdateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", formatValidationError(err))
		return
	}

	svc, err := h.repo.GetServiceByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Service not found")
		return
	}

	providerID := middleware.GetProviderID(r.Context())
	if svc.ProviderID != providerID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Not your service")
		return
	}

	svc.Name = req.Name
	svc.Description = req.Description
	svc.DurationMinutes = req.DurationMinutes
	svc.IsActive = req.IsActive

	if err := h.repo.UpdateService(r.Context(), svc); err != nil {
		slog.Error("provider.handler: update service: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Update failed")
		return
	}

	writeJSON(w, http.StatusOK, svc)
}

// DeleteService handles DELETE /v1/services/{id}.
func (h *Handler) DeleteService(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid service ID")
		return
	}

	// Verify ownership
	svc, err := h.repo.GetServiceByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Service not found")
		return
	}

	providerID := middleware.GetProviderID(r.Context())
	if svc.ProviderID != providerID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Not your service")
		return
	}

	if err := h.repo.DeleteService(r.Context(), id); err != nil {
		slog.Error("provider.handler: delete service: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Delete failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
