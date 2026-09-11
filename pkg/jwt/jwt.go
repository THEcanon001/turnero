package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims holds the custom JWT claims for the Turnero app.
type Claims struct {
	UserID     uuid.UUID `json:"user_id"`
	ProviderID uuid.UUID `json:"provider_id"`
	Role       string    `json:"role"` // "admin" or "employee"
	TokenType  string    `json:"token_type"`
	jwt.RegisteredClaims
}

// Config holds JWT signing configuration.
type Config struct {
	Secret             string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	Issuer             string
}

// Manager handles JWT token generation and validation.
type Manager struct {
	cfg Config
}

// NewManager creates a new JWT manager.
func NewManager(cfg Config) *Manager {
	return &Manager{cfg: cfg}
}

// GenerateAccessToken creates a signed access token.
func (m *Manager) GenerateAccessToken(userID, providerID uuid.UUID, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:     userID,
		ProviderID: providerID,
		Role:       role,
		TokenType:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.cfg.AccessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    m.cfg.Issuer,
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(m.cfg.Secret))
	if err != nil {
		return "", fmt.Errorf("jwt: sign access token: %w", err)
	}
	return signed, nil
}

// GenerateRefreshToken creates a signed refresh token.
func (m *Manager) GenerateRefreshToken(userID, providerID uuid.UUID, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:     userID,
		ProviderID: providerID,
		Role:       role,
		TokenType:  "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.cfg.RefreshTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    m.cfg.Issuer,
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(m.cfg.Secret))
	if err != nil {
		return "", fmt.Errorf("jwt: sign refresh token: %w", err)
	}
	return signed, nil
}

// ValidateToken parses and validates a JWT token string, returning the claims.
func (m *Manager) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("jwt: unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(m.cfg.Secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("jwt: token expired: %w", err)
		}
		return nil, fmt.Errorf("jwt: invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("jwt: invalid token claims")
	}

	return claims, nil
}
