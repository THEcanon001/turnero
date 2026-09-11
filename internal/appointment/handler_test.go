package appointment_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/THEcanon001/turnero/internal/appointment"
)

func TestCreate_InvalidBody(t *testing.T) {
	h := appointment.NewHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/appointments", bytes.NewReader([]byte("bad")))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreate_ValidationError(t *testing.T) {
	h := appointment.NewHandler(nil, nil, nil, nil)

	tests := []struct {
		name string
		body map[string]any
	}{
		{
			name: "missing client_name",
			body: map[string]any{
				"provider_slug": "test",
				"employee_id":   "00000000-0000-0000-0000-000000000001",
				"client_phone":  "+521234567890",
				"date":          "2026-09-15",
				"start_time":    "09:00",
			},
		},
		{
			name: "missing provider_slug",
			body: map[string]any{
				"employee_id":  "00000000-0000-0000-0000-000000000001",
				"client_name":  "Test",
				"client_phone": "+521234567890",
				"date":         "2026-09-15",
				"start_time":   "09:00",
			},
		},
		{
			name: "missing date",
			body: map[string]any{
				"provider_slug": "test",
				"employee_id":   "00000000-0000-0000-0000-000000000001",
				"client_name":   "Test",
				"client_phone":  "+521234567890",
				"start_time":    "09:00",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/v1/appointments", bytes.NewReader(body))
			rec := httptest.NewRecorder()
			h.Create(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestGetByID_InvalidID(t *testing.T) {
	h := appointment.NewHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/appointments/not-uuid", nil)
	rec := httptest.NewRecorder()
	h.GetByID(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCancel_InvalidID(t *testing.T) {
	h := appointment.NewHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/appointments/not-uuid/cancel", nil)
	rec := httptest.NewRecorder()
	h.Cancel(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateStatus_InvalidID(t *testing.T) {
	h := appointment.NewHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/appointments/not-uuid/status", nil)
	rec := httptest.NewRecorder()
	h.UpdateStatus(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateStatus_InvalidBody(t *testing.T) {
	h := appointment.NewHandler(nil, nil, nil, nil)

	// Valid UUID but bad body
	req := httptest.NewRequest(http.MethodPut, "/v1/appointments/00000000-0000-0000-0000-000000000001/status",
		bytes.NewReader([]byte("bad")))
	rec := httptest.NewRecorder()
	// Without chi context, UUID param is empty → 400
	h.UpdateStatus(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetSlots_InvalidEmployeeID(t *testing.T) {
	h := appointment.NewHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/providers/test/employees/not-uuid/slots?date=2026-09-15", nil)
	rec := httptest.NewRecorder()
	h.GetSlots(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetProviderProfile_EmptySlug(t *testing.T) {
	h := appointment.NewHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/providers/", nil)
	rec := httptest.NewRecorder()
	h.GetProviderProfile(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestReschedule_InvalidID(t *testing.T) {
	h := appointment.NewHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/appointments/not-uuid/reschedule", nil)
	rec := httptest.NewRecorder()
	h.Reschedule(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestReschedule_InvalidBody(t *testing.T) {
	h := appointment.NewHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/appointments/00000000-0000-0000-0000-000000000001/reschedule",
		bytes.NewReader([]byte("bad")))
	rec := httptest.NewRecorder()
	h.Reschedule(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestReschedule_ValidationError(t *testing.T) {
	h := appointment.NewHandler(nil, nil, nil, nil)

	body, _ := json.Marshal(map[string]string{"date": "2026-09-15"}) // missing start_time
	req := httptest.NewRequest(http.MethodPost, "/v1/appointments/00000000-0000-0000-0000-000000000001/reschedule",
		bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Reschedule(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestProviderCancel_InvalidID(t *testing.T) {
	h := appointment.NewHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/appointments/not-uuid/provider-cancel", nil)
	rec := httptest.NewRecorder()
	h.ProviderCancel(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestReassign_InvalidID(t *testing.T) {
	h := appointment.NewHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/appointments/not-uuid/reassign", nil)
	rec := httptest.NewRecorder()
	h.Reassign(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestReassign_InvalidBody(t *testing.T) {
	h := appointment.NewHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/appointments/00000000-0000-0000-0000-000000000001/reassign",
		bytes.NewReader([]byte("bad")))
	rec := httptest.NewRecorder()
	h.Reassign(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWalkIn_InvalidBody(t *testing.T) {
	h := appointment.NewHandler(nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/appointments/walk-in",
		bytes.NewReader([]byte("bad")))
	rec := httptest.NewRecorder()
	h.WalkIn(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWalkIn_ValidationError(t *testing.T) {
	h := appointment.NewHandler(nil, nil, nil, nil)

	body, _ := json.Marshal(map[string]any{
		"client_name": "Test",
		"client_phone": "123",
	}) // missing required fields
	req := httptest.NewRequest(http.MethodPost, "/v1/appointments/walk-in",
		bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.WalkIn(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
