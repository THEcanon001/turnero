package billing

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCalculateBillable(t *testing.T) {
	tests := []struct {
		name          string
		completed     int
		freeTierLimit int
		expected      int
	}{
		{"zero completed", 0, 20, 0},
		{"under free tier", 10, 20, 0},
		{"exactly at free tier", 20, 20, 0},
		{"one over free tier", 21, 20, 1},
		{"well over free tier", 50, 20, 30},
		{"zero free tier", 5, 0, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateBillable(tt.completed, tt.freeTierLimit)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateAmountDue(t *testing.T) {
	tests := []struct {
		name     string
		billable int
		expected float64
	}{
		{"zero billable", 0, 0.00},
		{"one billable", 1, 0.50},
		{"ten billable", 10, 5.00},
		{"twenty billable - at cap", 20, 10.00},
		{"thirty billable - capped", 30, 10.00},
		{"hundred billable - capped", 100, 10.00},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateAmountDue(tt.billable)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBillingUsage_ToUsageResponse(t *testing.T) {
	u := &BillingUsage{
		ID:                    uuid.New(),
		ProviderID:            uuid.New(),
		Month:                 time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		CompletedAppointments: 25,
		FreeTierLimit:         20,
		BillableAppointments:  5,
		AmountDue:             2.50,
		IsPaid:                false,
	}

	resp := u.ToUsageResponse()

	assert.Equal(t, 25, resp.CompletedAppointments)
	assert.Equal(t, 20, resp.FreeTierLimit)
	assert.Equal(t, 5, resp.BillableAppointments)
	assert.Equal(t, 2.50, resp.AmountDue)
	assert.False(t, resp.IsPaid)
	assert.Equal(t, "2026-09", resp.Month)
}

func TestBillingTransaction_ToTransactionResponse(t *testing.T) {
	desc := "Monthly billing"
	extID := "pay_123"
	tx := &BillingTransaction{
		ID:                uuid.New(),
		ProviderID:        uuid.New(),
		Amount:            5.00,
		Currency:          "MXN",
		Description:       &desc,
		ExternalPaymentID: &extID,
		CreatedAt:         time.Now(),
	}

	resp := tx.ToTransactionResponse()

	assert.Equal(t, tx.ID, resp.ID)
	assert.Equal(t, 5.00, resp.Amount)
	assert.Equal(t, "MXN", resp.Currency)
	assert.Equal(t, &desc, resp.Description)
	assert.Equal(t, &extID, resp.ExternalPaymentID)
}

func TestBillingTransaction_ToTransactionResponse_NilOptionals(t *testing.T) {
	tx := &BillingTransaction{
		ID:         uuid.New(),
		ProviderID: uuid.New(),
		Amount:     10.00,
		Currency:   "MXN",
		CreatedAt:  time.Now(),
	}

	resp := tx.ToTransactionResponse()

	assert.Nil(t, resp.Description)
	assert.Nil(t, resp.ExternalPaymentID)
}

func TestConstants(t *testing.T) {
	assert.Equal(t, 20, FreeTierLimit)
	assert.Equal(t, 0.50, PricePerAppointment)
	assert.Equal(t, 10.00, MonthlyCap)
	assert.Equal(t, "MXN", Currency)
	assert.Equal(t, 3, GraceThreshold)
}

func TestCalculateAmountDue_ExactCap(t *testing.T) {
	// 20 billable * 0.50 = 10.00, exactly at cap
	assert.Equal(t, 10.00, CalculateAmountDue(20))
	// 19 billable * 0.50 = 9.50, under cap
	assert.Equal(t, 9.50, CalculateAmountDue(19))
}
