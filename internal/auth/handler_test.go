package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/THEcanon001/turnero/internal/auth"
)

func TestNewHandler_SlugValidation(t *testing.T) {
	// We can't test the full flow without a DB, but we can test validation.
	// Create handler with nil service - it will fail at service level,
	// but we test that validation runs before service is called.
	handler := auth.NewHandler(nil)

	tests := []struct {
		name       string
		body       map[string]any
		wantStatus int
		wantCode   string
	}{
		{
			name:       "missing required fields",
			body:       map[string]any{},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name: "invalid email",
			body: map[string]any{
				"name":     "Test User",
				"email":    "not-an-email",
				"password": "password123",
				"phone":    "+521234567890",
				"type":     "individual",
				"slug":     "test-slug",
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name: "password too short",
			body: map[string]any{
				"name":     "Test User",
				"email":    "test@example.com",
				"password": "short",
				"phone":    "+521234567890",
				"type":     "individual",
				"slug":     "test-slug",
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name: "invalid type",
			body: map[string]any{
				"name":     "Test User",
				"email":    "test@example.com",
				"password": "password123",
				"phone":    "+521234567890",
				"type":     "invalid",
				"slug":     "test-slug",
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name: "slug with uppercase",
			body: map[string]any{
				"name":     "Test User",
				"email":    "test@example.com",
				"password": "password123",
				"phone":    "+521234567890",
				"type":     "individual",
				"slug":     "Bad-Slug",
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name: "slug too short",
			body: map[string]any{
				"name":     "Test User",
				"email":    "test@example.com",
				"password": "password123",
				"phone":    "+521234567890",
				"type":     "individual",
				"slug":     "ab",
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
		{
			name: "business without business_name",
			body: map[string]any{
				"name":     "Test Business",
				"email":    "biz@example.com",
				"password": "password123",
				"phone":    "+521234567890",
				"type":     "business",
				"slug":     "my-business",
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.Register(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			var resp map[string]string
			json.Unmarshal(rec.Body.Bytes(), &resp)
			assert.Equal(t, tt.wantCode, resp["code"])
		})
	}
}

func TestLogin_InvalidBody(t *testing.T) {
	handler := auth.NewHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader([]byte("not json")))
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLogin_MissingFields(t *testing.T) {
	handler := auth.NewHandler(nil)

	body, _ := json.Marshal(map[string]any{"email": "test@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRefresh_InvalidBody(t *testing.T) {
	handler := auth.NewHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", bytes.NewReader([]byte("{}")))
	rec := httptest.NewRecorder()

	handler.Refresh(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestLogout_InvalidBody(t *testing.T) {
	handler := auth.NewHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", bytes.NewReader([]byte("bad")))
	rec := httptest.NewRecorder()

	handler.Logout(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegister_InvalidBody(t *testing.T) {
	handler := auth.NewHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader([]byte("bad json")))
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
