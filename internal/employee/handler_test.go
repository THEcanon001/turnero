package employee_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/THEcanon001/turnero/internal/employee"
)

func TestList_NilRepo(t *testing.T) {
	// Handler with nil repo will panic — verifies handler wiring
	h := employee.NewHandler(nil)
	assert.NotNil(t, h)
}

func TestCreate_InvalidBody(t *testing.T) {
	h := employee.NewHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/employees", bytes.NewReader([]byte("invalid")))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "VALIDATION_ERROR")
}

func TestCreate_ValidationError(t *testing.T) {
	h := employee.NewHandler(nil)

	tests := []struct {
		name string
		body map[string]any
	}{
		{"missing name", map[string]any{"phone": "123", "role": "employee"}},
		{"missing phone", map[string]any{"name": "John", "role": "employee"}},
		{"missing role", map[string]any{"name": "John", "phone": "123"}},
		{"invalid role", map[string]any{"name": "John", "phone": "123", "role": "superuser"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/v1/employees", bytes.NewReader(body))
			rec := httptest.NewRecorder()
			h.Create(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestUpdate_InvalidID(t *testing.T) {
	h := employee.NewHandler(nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/employees/bad-id", nil)
	rec := httptest.NewRecorder()
	h.Update(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDelete_InvalidID(t *testing.T) {
	h := employee.NewHandler(nil)

	req := httptest.NewRequest(http.MethodDelete, "/v1/employees/bad-id", nil)
	rec := httptest.NewRecorder()
	h.Delete(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetByID_InvalidID(t *testing.T) {
	h := employee.NewHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/employees/bad-id", nil)
	rec := httptest.NewRecorder()
	h.GetByID(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateInvitation_InvalidBody(t *testing.T) {
	h := employee.NewHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/employees/invitations", bytes.NewReader([]byte("invalid")))
	rec := httptest.NewRecorder()
	h.CreateInvitation(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateInvitation_ValidationError(t *testing.T) {
	h := employee.NewHandler(nil)

	body, _ := json.Marshal(map[string]string{"employee_name": "A"}) // too short
	req := httptest.NewRequest(http.MethodPost, "/v1/employees/invitations", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.CreateInvitation(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAssignServices_InvalidID(t *testing.T) {
	h := employee.NewHandler(nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/employees/bad-id/services", nil)
	rec := httptest.NewRecorder()
	h.AssignServices(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAssignServices_InvalidBody(t *testing.T) {
	h := employee.NewHandler(nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/employees/00000000-0000-0000-0000-000000000001/services",
		bytes.NewReader([]byte("invalid")))
	rec := httptest.NewRecorder()
	h.AssignServices(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdate_InvalidBody(t *testing.T) {
	h := employee.NewHandler(nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/employees/00000000-0000-0000-0000-000000000001",
		bytes.NewReader([]byte("invalid")))
	rec := httptest.NewRecorder()
	h.Update(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdate_ValidationError(t *testing.T) {
	h := employee.NewHandler(nil)

	body, _ := json.Marshal(map[string]string{"name": "A"}) // missing phone
	req := httptest.NewRequest(http.MethodPut, "/v1/employees/00000000-0000-0000-0000-000000000001",
		bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Update(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
