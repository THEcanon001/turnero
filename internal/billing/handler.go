package billing

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/THEcanon001/turnero/internal/platform/middleware"
)

// Handler handles HTTP requests for billing endpoints.
type Handler struct {
	service *Service
	repo    *Repository
}

// NewHandler creates a new billing handler.
func NewHandler(service *Service, repo *Repository) *Handler {
	return &Handler{
		service: service,
		repo:    repo,
	}
}

// GetCurrentUsage handles GET /v1/billing/usage.
func (h *Handler) GetCurrentUsage(w http.ResponseWriter, r *http.Request) {
	providerID := middleware.GetProviderID(r.Context())

	usage, err := h.service.GetCurrentUsage(r.Context(), providerID)
	if err != nil {
		slog.Error(fmt.Sprintf("billing.handler: GetCurrentUsage: %s", err.Error()))
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get billing usage")
		return
	}

	writeJSON(w, http.StatusOK, usage.ToUsageResponse())
}

// GetUsageHistory handles GET /v1/billing/history?limit=12.
func (h *Handler) GetUsageHistory(w http.ResponseWriter, r *http.Request) {
	providerID := middleware.GetProviderID(r.Context())

	limit := 12
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	usages, err := h.repo.ListUsageHistory(r.Context(), providerID, limit)
	if err != nil {
		slog.Error(fmt.Sprintf("billing.handler: GetUsageHistory: %s", err.Error()))
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get billing history")
		return
	}

	resp := make([]UsageResponse, len(usages))
	for i, u := range usages {
		resp[i] = u.ToUsageResponse()
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetTransactions handles GET /v1/billing/transactions?limit=20.
func (h *Handler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	providerID := middleware.GetProviderID(r.Context())

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	txns, err := h.repo.ListTransactions(r.Context(), providerID, limit)
	if err != nil {
		slog.Error(fmt.Sprintf("billing.handler: GetTransactions: %s", err.Error()))
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get transactions")
		return
	}

	resp := make([]TransactionResponse, len(txns))
	for i, tx := range txns {
		resp[i] = tx.ToTransactionResponse()
	}

	writeJSON(w, http.StatusOK, resp)
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
