package qr_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/THEcanon001/turnero/internal/qr"
)

func TestGetQR_NoProviderID(t *testing.T) {
	s := qr.NewService("https://turnero.app", t.TempDir())
	// nil providerRepo — handler will fail when trying to look up provider
	h := qr.NewHandler(s, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/provider/me/qr", nil)
	rec := httptest.NewRecorder()

	// This will panic because providerRepo is nil and we haven't set provider ID in context,
	// but we use recovery to verify the handler exists and wires correctly.
	require.Panics(t, func() {
		h.GetQR(rec, req)
	})
}

func TestNewHandler(t *testing.T) {
	s := qr.NewService("https://turnero.app", t.TempDir())
	h := qr.NewHandler(s, nil)
	assert.NotNil(t, h)
}
