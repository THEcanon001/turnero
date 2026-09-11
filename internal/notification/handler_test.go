package notification_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/THEcanon001/turnero/internal/notification"
)

func TestRegisterToken_InvalidBody(t *testing.T) {
	h := notification.NewHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/push-tokens", bytes.NewReader([]byte("bad")))
	rec := httptest.NewRecorder()
	h.RegisterToken(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegisterToken_ValidationError(t *testing.T) {
	h := notification.NewHandler(nil)

	tests := []struct {
		name string
		body map[string]any
	}{
		{"missing token", map[string]any{"platform": "ios"}},
		{"missing platform", map[string]any{"token": "abc123"}},
		{"invalid platform", map[string]any{"token": "abc123", "platform": "blackberry"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/v1/push-tokens", bytes.NewReader(body))
			rec := httptest.NewRecorder()
			h.RegisterToken(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestDeregisterToken_InvalidBody(t *testing.T) {
	h := notification.NewHandler(nil)

	req := httptest.NewRequest(http.MethodDelete, "/v1/push-tokens", bytes.NewReader([]byte("bad")))
	rec := httptest.NewRecorder()
	h.DeregisterToken(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeregisterToken_MissingToken(t *testing.T) {
	h := notification.NewHandler(nil)

	body, _ := json.Marshal(map[string]string{"token": ""})
	req := httptest.NewRequest(http.MethodDelete, "/v1/push-tokens", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.DeregisterToken(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestNewHandler(t *testing.T) {
	h := notification.NewHandler(nil)
	assert.NotNil(t, h)
}
