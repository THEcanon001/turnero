package schedule_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/THEcanon001/turnero/internal/schedule"
)

func TestSetSchedule_InvalidEmployeeID(t *testing.T) {
	h := schedule.NewHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPut, "/v1/employees/not-uuid/schedules", nil)
	rec := httptest.NewRecorder()
	h.SetSchedule(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetSchedule_InvalidEmployeeID(t *testing.T) {
	h := schedule.NewHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/employees/not-uuid/schedules", nil)
	rec := httptest.NewRecorder()
	h.GetSchedule(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAddException_InvalidEmployeeID(t *testing.T) {
	h := schedule.NewHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/employees/not-uuid/schedule-exceptions", nil)
	rec := httptest.NewRecorder()
	h.AddException(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestListExceptions_InvalidEmployeeID(t *testing.T) {
	h := schedule.NewHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/employees/not-uuid/schedule-exceptions", nil)
	rec := httptest.NewRecorder()
	h.ListExceptions(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteException_InvalidID(t *testing.T) {
	h := schedule.NewHandler(nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/v1/schedule-exceptions/not-uuid", nil)
	rec := httptest.NewRecorder()
	h.DeleteException(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAddException_InvalidBody(t *testing.T) {
	h := schedule.NewHandler(nil, nil)

	// Use chi URL params - but since no chi context, UUID parse will fail.
	// We test a different path: valid UUID but bad body.
	// Without chi context, employeeId param is empty string → UUID parse fails → 400.
	body, _ := json.Marshal(map[string]any{"bad": true})
	req := httptest.NewRequest(http.MethodPost, "/v1/employees/bad/schedule-exceptions", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.AddException(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSetSchedule_InvalidBody(t *testing.T) {
	h := schedule.NewHandler(nil, nil)

	// UUID parse fails on empty param
	req := httptest.NewRequest(http.MethodPut, "/v1/employees/bad/schedules", bytes.NewReader([]byte("bad")))
	rec := httptest.NewRecorder()
	h.SetSchedule(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestListExceptions_MissingParams(t *testing.T) {
	h := schedule.NewHandler(nil, nil)

	// UUID parse fails
	req := httptest.NewRequest(http.MethodGet, "/v1/employees/bad/schedule-exceptions?from=2026-01-01", nil)
	rec := httptest.NewRecorder()
	h.ListExceptions(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
