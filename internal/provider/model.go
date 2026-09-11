package provider

import (
	"time"

	"github.com/google/uuid"
)

type Type string

const (
	TypeIndividual Type = "individual"
	TypeBusiness   Type = "business"
)

type BillingPlan string

const (
	PlanFree  BillingPlan = "free"
	PlanBasic BillingPlan = "basic"
	PlanPro   BillingPlan = "pro"
)

// Provider represents a service provider (PF or PJ).
type Provider struct {
	ID                uuid.UUID   `json:"id"`
	Type              Type        `json:"type"`
	Name              string      `json:"name"`
	Slug              string      `json:"slug"`
	Phone             string      `json:"phone"`
	Address           *string     `json:"address,omitempty"`
	Timezone          string      `json:"timezone"`
	BusinessName      *string     `json:"business_name,omitempty"`
	Email             string      `json:"email"`
	PasswordHash      string      `json:"-"`
	QRImagePath       *string     `json:"qr_image_path,omitempty"`
	Plan              BillingPlan `json:"plan"`
	BillingCycleStart *time.Time  `json:"billing_cycle_start,omitempty"`
	IsActive          bool        `json:"is_active"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}
