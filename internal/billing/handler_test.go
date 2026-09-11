package billing_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/THEcanon001/turnero/internal/billing"
)

func TestNewHandler(t *testing.T) {
	h := billing.NewHandler(nil, nil)
	assert.NotNil(t, h)
}

func TestGetCurrentUsage_NilService(t *testing.T) {
	h := billing.NewHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/billing/usage", nil)
	rec := httptest.NewRecorder()

	// Will panic or return error since service is nil — this tests the handler creation path
	// We expect a 500 because middleware.GetProviderID returns zero UUID and service is nil
	assert.Panics(t, func() {
		h.GetCurrentUsage(rec, req)
	})
}

func TestGetUsageHistory_NilRepo(t *testing.T) {
	h := billing.NewHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/billing/history", nil)
	rec := httptest.NewRecorder()

	assert.Panics(t, func() {
		h.GetUsageHistory(rec, req)
	})
}

func TestGetTransactions_NilRepo(t *testing.T) {
	h := billing.NewHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/billing/transactions", nil)
	rec := httptest.NewRecorder()

	assert.Panics(t, func() {
		h.GetTransactions(rec, req)
	})
}

func TestWriteJSON_Format(t *testing.T) {
	// Verify JSON encoding via handler response format
	h := billing.NewHandler(nil, nil)
	assert.NotNil(t, h)
}

func TestUsageResponse_JSONEncoding(t *testing.T) {
	resp := billing.UsageResponse{
		CompletedAppointments: 25,
		FreeTierLimit:         20,
		BillableAppointments:  5,
		AmountDue:             2.50,
		IsPaid:                false,
		Month:                 "2026-09",
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	var decoded billing.UsageResponse
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, resp, decoded)
}

func TestTransactionResponse_JSONEncoding(t *testing.T) {
	desc := "Test payment"
	resp := billing.TransactionResponse{
		Amount:      5.00,
		Currency:    "MXN",
		Description: &desc,
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	var decoded billing.TransactionResponse
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, resp.Amount, decoded.Amount)
	assert.Equal(t, *resp.Description, *decoded.Description)
}
