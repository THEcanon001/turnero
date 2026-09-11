package search

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Result represents a search result entry.
type Result struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Slug    string  `json:"slug"`
	Phone   string  `json:"phone"`
	Address *string `json:"address,omitempty"`
	Type    string  `json:"type"`
}

// Handler handles search HTTP requests.
type Handler struct {
	pool *pgxpool.Pool
}

// NewHandler creates a new search handler.
func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

// Search handles GET /v1/search?q=query.
// Public endpoint — fuzzy search on provider names using pg_trgm.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" || len(q) < 2 {
		writeJSON(w, http.StatusOK, []Result{})
		return
	}

	query := `
		SELECT id::text, name, slug, phone, address, type::text
		FROM providers
		WHERE is_active = true AND (
			name ILIKE '%' || $1 || '%'
			OR slug ILIKE '%' || $1 || '%'
		)
		ORDER BY similarity(name, $1) DESC
		LIMIT 20`

	rows, err := h.pool.Query(r.Context(), query, q)
	if err != nil {
		slog.Error("search.handler: query: " + err.Error())
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Search failed")
		return
	}
	defer rows.Close()

	var results []Result
	for rows.Next() {
		var res Result
		if err := rows.Scan(&res.ID, &res.Name, &res.Slug, &res.Phone, &res.Address, &res.Type); err != nil {
			slog.Error("search.handler: scan: " + err.Error())
			continue
		}
		results = append(results, res)
	}

	if results == nil {
		results = []Result{}
	}
	writeJSON(w, http.StatusOK, results)
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
