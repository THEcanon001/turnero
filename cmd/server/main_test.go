package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDB struct {
	pingErr error
}

func (m *mockDB) Ping(_ context.Context) error {
	return m.pingErr
}

func TestHealthHandler_OK(t *testing.T) {
	handler := healthHandler(&mockDB{pingErr: nil})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	require.NoError(t, err)

	assert.Equal(t, "ok", body["status"])
	assert.Equal(t, "ok", body["database"])
}

func TestHealthHandler_DBDown(t *testing.T) {
	handler := healthHandler(&mockDB{pingErr: errors.New("connection refused")})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var body map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	require.NoError(t, err)

	assert.Equal(t, "degraded", body["status"])
	assert.Equal(t, "error", body["database"])
}
