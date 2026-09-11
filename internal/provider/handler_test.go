package provider_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/THEcanon001/turnero/internal/provider"
)

func TestCreateService_InvalidBody(t *testing.T) {
	h := provider.NewHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/services", bytes.NewReader([]byte("bad")))
	rec := httptest.NewRecorder()
	h.CreateService(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateService_ValidationErrors(t *testing.T) {
	h := provider.NewHandler(nil)

	tests := []struct {
		name string
		body map[string]any
	}{
		{
			name: "missing name",
			body: map[string]any{"duration_minutes": 30},
		},
		{
			name: "duration too short",
			body: map[string]any{"name": "Test", "duration_minutes": 2},
		},
		{
			name: "duration too long",
			body: map[string]any{"name": "Test", "duration_minutes": 500},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/v1/services", bytes.NewReader(body))
			rec := httptest.NewRecorder()
			h.CreateService(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestUpdateService_InvalidBody(t *testing.T) {
	h := provider.NewHandler(nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/services/not-uuid", bytes.NewReader([]byte("bad")))
	rec := httptest.NewRecorder()

	// Need chi context for URL params - test the invalid UUID path
	h.UpdateService(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetService_InvalidID(t *testing.T) {
	h := provider.NewHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/services/not-uuid", nil)
	rec := httptest.NewRecorder()
	h.GetService(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteService_InvalidID(t *testing.T) {
	h := provider.NewHandler(nil)

	req := httptest.NewRequest(http.MethodDelete, "/v1/services/not-uuid", nil)
	rec := httptest.NewRecorder()
	h.DeleteService(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateMe_InvalidBody(t *testing.T) {
	h := provider.NewHandler(nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/provider/me", bytes.NewReader([]byte("bad json")))
	rec := httptest.NewRecorder()
	h.UpdateMe(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateMe_ValidationError(t *testing.T) {
	h := provider.NewHandler(nil)

	body, _ := json.Marshal(map[string]any{"name": "A"}) // name too short + missing phone/timezone
	req := httptest.NewRequest(http.MethodPut, "/v1/provider/me", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.UpdateMe(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
