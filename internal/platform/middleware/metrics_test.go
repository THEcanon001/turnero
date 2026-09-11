package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"github.com/THEcanon001/turnero/internal/platform/middleware"
)

func TestMetrics_MiddlewareReturnsHandler(t *testing.T) {
	mw := middleware.Metrics()
	assert.NotNil(t, mw)
}

func TestMetrics_TracksStatusCode(t *testing.T) {
	r := chi.NewRouter()
	r.Use(middleware.Metrics())
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestMetrics_Tracks404(t *testing.T) {
	r := chi.NewRouter()
	r.Use(middleware.Metrics())
	r.Get("/exists", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/not-exists", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestMetrics_TracksDifferentMethods(t *testing.T) {
	r := chi.NewRouter()
	r.Use(middleware.Metrics())
	r.Post("/items", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	r.Get("/items", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// POST
	req := httptest.NewRequest(http.MethodPost, "/items", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)

	// GET
	req = httptest.NewRequest(http.MethodGet, "/items", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMetrics_Tracks500(t *testing.T) {
	r := chi.NewRouter()
	r.Use(middleware.Metrics())
	r.Get("/error", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestMetrics_DefaultStatus200(t *testing.T) {
	r := chi.NewRouter()
	r.Use(middleware.Metrics())
	r.Get("/default", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})

	req := httptest.NewRequest(http.MethodGet, "/default", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}
