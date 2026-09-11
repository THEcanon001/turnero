package notification

import (
	"time"

	"github.com/google/uuid"
)

// PushToken represents a registered device token for push notifications.
type PushToken struct {
	ID          uuid.UUID  `json:"id"`
	ProviderID  *uuid.UUID `json:"provider_id,omitempty"`
	EmployeeID  *uuid.UUID `json:"employee_id,omitempty"`
	ClientPhone *string    `json:"client_phone,omitempty"`
	Token       string     `json:"token"`
	Platform    string     `json:"platform"` // ios, android, web
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// RegisterTokenRequest is the input for registering a push token.
type RegisterTokenRequest struct {
	Token    string `json:"token" validate:"required"`
	Platform string `json:"platform" validate:"required,oneof=ios android web"`
}

// Message represents a push notification message.
type Message struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Data  map[string]string `json:"data,omitempty"`
}
