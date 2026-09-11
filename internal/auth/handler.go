package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Handler handles HTTP requests for authentication.
type Handler struct {
	service        *Service
	validate       *validator.Validate
	googleClientID string
}

// NewHandler creates a new auth handler.
func NewHandler(service *Service, googleClientID ...string) *Handler {
	v := validator.New()
	// Custom slug validation: lowercase letters, numbers, hyphens only
	v.RegisterValidation("slug", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		for _, c := range s {
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
				return false
			}
		}
		return true
	})

	var gClientID string
	if len(googleClientID) > 0 {
		gClientID = googleClientID[0]
	}

	return &Handler{
		service:        service,
		validate:       v,
		googleClientID: gClientID,
	}
}

// Register handles POST /v1/auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", formatValidationError(err))
		return
	}

	// Enforce business_name for business type
	if req.Type == "business" && (req.BusinessName == nil || *req.BusinessName == "") {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "business_name is required for business type")
		return
	}

	resp, err := h.service.Register(r.Context(), req)
	if err != nil {
		slog.Error("auth.handler: register: " + err.Error())
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique") {
			writeError(w, http.StatusConflict, "CONFLICT", "Email or slug already taken")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Registration failed")
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// Login handles POST /v1/auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", formatValidationError(err))
		return
	}

	resp, err := h.service.Login(r.Context(), req)
	if err != nil {
		slog.Warn("auth.handler: login: " + err.Error())
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid email or password")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// Refresh handles POST /v1/auth/refresh.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", formatValidationError(err))
		return
	}

	resp, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		slog.Warn("auth.handler: refresh: " + err.Error())
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired refresh token")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// Logout handles POST /v1/auth/logout.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if err := h.service.Logout(r.Context(), req.RefreshToken); err != nil {
		slog.Warn("auth.handler: logout: " + err.Error())
	}

	w.WriteHeader(http.StatusNoContent)
}

// Join handles POST /v1/auth/join.
func (h *Handler) Join(w http.ResponseWriter, r *http.Request) {
	var req JoinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", formatValidationError(err))
		return
	}

	resp, err := h.service.Join(r.Context(), req)
	if err != nil {
		slog.Warn("auth.handler: join: " + err.Error())
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "expired") {
			writeError(w, http.StatusBadRequest, "INVALID_CODE", "Invalid or expired invitation code")
			return
		}
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique") {
			writeError(w, http.StatusConflict, "CONFLICT", "Email already in use")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Join failed")
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// GoogleLogin handles POST /v1/auth/google.
func (h *Handler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var req GoogleLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", formatValidationError(err))
		return
	}

	if req.Type == "business" && (req.BusinessName == nil || *req.BusinessName == "") {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "business_name is required for business type")
		return
	}

	// Verify Google id_token with audience validation
	claims, err := VerifyGoogleToken(r.Context(), req.IDToken, h.googleClientID)
	if err != nil {
		slog.Warn("auth.handler: google login: " + err.Error())
		writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid Google token")
		return
	}

	resp, err := h.service.GoogleLogin(r.Context(), req, claims)
	if err != nil {
		slog.Error("auth.handler: google login: " + err.Error())
		if strings.Contains(err.Error(), "is required for new Google accounts") {
			writeError(w, http.StatusBadRequest, "MISSING_FIELDS", err.Error())
			return
		}
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique") {
			writeError(w, http.StatusConflict, "CONFLICT", "Slug already taken")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Google login failed")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// LoginEmployee handles POST /v1/auth/login/employee.
func (h *Handler) LoginEmployee(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", formatValidationError(err))
		return
	}

	resp, err := h.service.LoginEmployee(r.Context(), req)
	if err != nil {
		slog.Warn("auth.handler: login employee: " + err.Error())
		if strings.Contains(err.Error(), "deactivated") {
			writeError(w, http.StatusForbidden, "ACCOUNT_DEACTIVATED", "Account has been deactivated")
			return
		}
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid email or password")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes a JSON error response.
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
