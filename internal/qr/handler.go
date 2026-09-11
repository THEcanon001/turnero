package qr

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/THEcanon001/turnero/internal/platform/middleware"
	"github.com/THEcanon001/turnero/internal/provider"
)

// Handler handles QR code HTTP requests.
type Handler struct {
	service      *Service
	providerRepo *provider.Repository
}

// NewHandler creates a new QR handler.
func NewHandler(service *Service, providerRepo *provider.Repository) *Handler {
	return &Handler{
		service:      service,
		providerRepo: providerRepo,
	}
}

// GetQR handles GET /v1/provider/me/qr.
// Returns the QR code as a PNG image.
func (h *Handler) GetQR(w http.ResponseWriter, r *http.Request) {
	providerID := middleware.GetProviderID(r.Context())

	p, err := h.providerRepo.GetByID(r.Context(), providerID)
	if err != nil {
		slog.Error("qr.handler: get provider: " + err.Error())
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Provider not found")
		return
	}

	png, err := h.service.GenerateBytes(p.Slug)
	if err != nil {
		slog.Error("qr.handler: generate: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate QR")
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", "inline; filename=\"qr-"+p.Slug+".png\"")
	w.WriteHeader(http.StatusOK)
	w.Write(png)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
		"code":  code,
	})
}
