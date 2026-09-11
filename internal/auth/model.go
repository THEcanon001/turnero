package auth

import (
	"time"

	"github.com/google/uuid"
)

// RegisterRequest is the input for provider registration.
type RegisterRequest struct {
	Name         string  `json:"name" validate:"required,min=2,max=100"`
	Email        string  `json:"email" validate:"required,email"`
	Password     string  `json:"password" validate:"required,min=8"`
	Phone        string  `json:"phone" validate:"required"`
	Type         string  `json:"type" validate:"required,oneof=individual business"`
	Slug         string  `json:"slug" validate:"required,min=3,max=50,slug"`
	BusinessName *string `json:"business_name"`
	Timezone     string  `json:"timezone"`
}

// LoginRequest is the input for login.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RefreshRequest is the input for token refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// LogoutRequest is the input for logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// TokenPair holds the access and refresh tokens.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// RegisterResponse is the output for registration.
type RegisterResponse struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Type         string    `json:"type"`
	Slug         string    `json:"slug"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
}

// LoginResponse is the output for login.
type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	User         LoginUser `json:"user"`
}

// LoginUser contains the user info returned on login.
type LoginUser struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Role       string    `json:"role"`
	Type       string    `json:"type"`
	ProviderID uuid.UUID `json:"provider_id"`
}

// RefreshToken represents a stored refresh token.
type RefreshToken struct {
	ID         uuid.UUID  `json:"id"`
	ProviderID *uuid.UUID `json:"provider_id,omitempty"`
	EmployeeID *uuid.UUID `json:"employee_id,omitempty"`
	TokenHash  string     `json:"-"`
	ExpiresAt  time.Time  `json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	ReplacedBy *uuid.UUID `json:"replaced_by,omitempty"`
	UserAgent  string     `json:"user_agent,omitempty"`
	IPAddress  string     `json:"ip_address,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}
