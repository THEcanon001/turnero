package billing

import (
	"time"

	"github.com/google/uuid"
)

// Billing plan constants
const (
	FreeTierLimit       = 20
	PricePerAppointment = 0.50
	MonthlyCap          = 10.00
	Currency            = "MXN"
	GraceThreshold      = 3 // consecutive unpaid months before restriction
)

// BillingUsage tracks monthly appointment usage and billing for a provider.
type BillingUsage struct {
	ID                    uuid.UUID `json:"id"`
	ProviderID            uuid.UUID `json:"provider_id"`
	Month                 time.Time `json:"month"`
	CompletedAppointments int       `json:"completed_appointments"`
	FreeTierLimit         int       `json:"free_tier_limit"`
	BillableAppointments  int       `json:"billable_appointments"`
	AmountDue             float64   `json:"amount_due"`
	IsPaid                bool      `json:"is_paid"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// BillingTransaction records a payment or charge event.
type BillingTransaction struct {
	ID                uuid.UUID `json:"id"`
	ProviderID        uuid.UUID `json:"provider_id"`
	BillingUsageID    *uuid.UUID `json:"billing_usage_id,omitempty"`
	Amount            float64   `json:"amount"`
	Currency          string    `json:"currency"`
	Description       *string   `json:"description,omitempty"`
	ExternalPaymentID *string   `json:"external_payment_id,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// UsageResponse is the API response for billing usage.
type UsageResponse struct {
	CompletedAppointments int     `json:"completed_appointments"`
	FreeTierLimit         int     `json:"free_tier_limit"`
	BillableAppointments  int     `json:"billable_appointments"`
	AmountDue             float64 `json:"amount_due"`
	IsPaid                bool    `json:"is_paid"`
	Month                 string  `json:"month"`
}

// TransactionResponse is the API response for a billing transaction.
type TransactionResponse struct {
	ID                uuid.UUID `json:"id"`
	Amount            float64   `json:"amount"`
	Currency          string    `json:"currency"`
	Description       *string   `json:"description,omitempty"`
	ExternalPaymentID *string   `json:"external_payment_id,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// CalculateBillable returns the number of billable appointments beyond the free tier.
func CalculateBillable(completed, freeTierLimit int) int {
	if completed <= freeTierLimit {
		return 0
	}
	return completed - freeTierLimit
}

// CalculateAmountDue returns the amount owed, capped at MonthlyCap.
func CalculateAmountDue(billable int) float64 {
	amount := float64(billable) * PricePerAppointment
	if amount > MonthlyCap {
		return MonthlyCap
	}
	return amount
}

// ToUsageResponse converts a BillingUsage to an API response.
func (u *BillingUsage) ToUsageResponse() UsageResponse {
	return UsageResponse{
		CompletedAppointments: u.CompletedAppointments,
		FreeTierLimit:         u.FreeTierLimit,
		BillableAppointments:  u.BillableAppointments,
		AmountDue:             u.AmountDue,
		IsPaid:                u.IsPaid,
		Month:                 u.Month.Format("2006-01"),
	}
}

// ToTransactionResponse converts a BillingTransaction to an API response.
func (t *BillingTransaction) ToTransactionResponse() TransactionResponse {
	return TransactionResponse{
		ID:                t.ID,
		Amount:            t.Amount,
		Currency:          t.Currency,
		Description:       t.Description,
		ExternalPaymentID: t.ExternalPaymentID,
		CreatedAt:         t.CreatedAt,
	}
}
