package notification

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/THEcanon001/turnero/internal/platform/middleware"
)

// Handler handles HTTP requests for push tokens.
type Handler struct {
	repo     *Repository
	validate *validator.Validate
}

// NewHandler creates a new notification handler.
func NewHandler(repo *Repository) *Handler {
	return &Handler{
		repo:     repo,
		validate: validator.New(),
	}
}

// RegisterToken handles POST /v1/push-tokens.
func (h *Handler) RegisterToken(w http.ResponseWriter, r *http.Request) {
	var req RegisterTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", formatValidationError(err))
		return
	}

	providerID := middleware.GetProviderID(r.Context())
	userID := middleware.GetUserID(r.Context())
	role := middleware.GetRole(r.Context())

	pt := &PushToken{
		Token:    req.Token,
		Platform: req.Platform,
	}

	if role == "admin" {
		pt.ProviderID = &providerID
	} else {
		pt.EmployeeID = &userID
	}

	if err := h.repo.RegisterToken(r.Context(), pt); err != nil {
		slog.Error("notification.handler: register token: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to register token")
		return
	}

	writeJSON(w, http.StatusCreated, pt)
}

// DeregisterToken handles DELETE /v1/push-tokens.
func (h *Handler) DeregisterToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "token is required")
		return
	}

	if err := h.repo.DeactivateToken(r.Context(), req.Token); err != nil {
		slog.Error("notification.handler: deregister token: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to deregister token")
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
