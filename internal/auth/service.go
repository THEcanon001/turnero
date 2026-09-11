package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/THEcanon001/turnero/internal/provider"
	tjwt "github.com/THEcanon001/turnero/pkg/jwt"
)

const bcryptCost = 12

// Service handles authentication business logic.
type Service struct {
	providerRepo *provider.Repository
	authRepo     *Repository
	jwtManager   *tjwt.Manager
	jwtCfg       tjwt.Config
}

// NewService creates a new auth service.
func NewService(
	providerRepo *provider.Repository,
	authRepo *Repository,
	jwtManager *tjwt.Manager,
	jwtCfg tjwt.Config,
) *Service {
	return &Service{
		providerRepo: providerRepo,
		authRepo:     authRepo,
		jwtManager:   jwtManager,
		jwtCfg:       jwtCfg,
	}
}

// Register creates a new provider and returns tokens.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("auth.service: hash password: %w", err)
	}

	tz := req.Timezone
	if tz == "" {
		tz = "America/Mexico_City"
	}

	p := &provider.Provider{
		Type:         provider.Type(req.Type),
		Name:         req.Name,
		Slug:         req.Slug,
		Phone:        req.Phone,
		Email:        req.Email,
		PasswordHash: string(hash),
		Timezone:     tz,
		BusinessName: req.BusinessName,
	}

	if err := s.providerRepo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("auth.service: %w", err)
	}

	// Generate tokens
	accessToken, err := s.jwtManager.GenerateAccessToken(p.ID, p.ID, "admin")
	if err != nil {
		return nil, fmt.Errorf("auth.service: %w", err)
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(p.ID, p.ID, "admin")
	if err != nil {
		return nil, fmt.Errorf("auth.service: %w", err)
	}

	// Store refresh token hash
	rt := &RefreshToken{
		ProviderID: &p.ID,
		TokenHash:  HashToken(refreshToken),
		ExpiresAt:  time.Now().Add(s.jwtCfg.RefreshTokenExpiry),
	}
	if err := s.authRepo.StoreRefreshToken(ctx, rt); err != nil {
		return nil, fmt.Errorf("auth.service: %w", err)
	}

	return &RegisterResponse{
		ID:           p.ID,
		Name:         p.Name,
		Email:        p.Email,
		Type:         string(p.Type),
		Slug:         p.Slug,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// Login authenticates a provider by email/password and returns tokens.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	p, err := s.providerRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("auth.service: invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(p.PasswordHash), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("auth.service: invalid credentials")
	}

	accessToken, err := s.jwtManager.GenerateAccessToken(p.ID, p.ID, "admin")
	if err != nil {
		return nil, fmt.Errorf("auth.service: %w", err)
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(p.ID, p.ID, "admin")
	if err != nil {
		return nil, fmt.Errorf("auth.service: %w", err)
	}

	rt := &RefreshToken{
		ProviderID: &p.ID,
		TokenHash:  HashToken(refreshToken),
		ExpiresAt:  time.Now().Add(s.jwtCfg.RefreshTokenExpiry),
	}
	if err := s.authRepo.StoreRefreshToken(ctx, rt); err != nil {
		return nil, fmt.Errorf("auth.service: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.jwtCfg.AccessTokenExpiry.Seconds()),
		User: LoginUser{
			ID:         p.ID,
			Name:       p.Name,
			Role:       "admin",
			Type:       string(p.Type),
			ProviderID: p.ID,
		},
	}, nil
}

// Refresh validates a refresh token, rotates it, and returns a new token pair.
func (s *Service) Refresh(ctx context.Context, rawRefreshToken string) (*TokenPair, error) {
	hash := HashToken(rawRefreshToken)

	storedToken, err := s.authRepo.GetByTokenHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("auth.service: invalid refresh token")
	}

	if time.Now().After(storedToken.ExpiresAt) {
		return nil, fmt.Errorf("auth.service: refresh token expired")
	}

	// Determine the user for the new token
	var userID, providerID uuid.UUID
	var role string
	if storedToken.ProviderID != nil {
		userID = *storedToken.ProviderID
		providerID = *storedToken.ProviderID
		role = "admin"
	} else if storedToken.EmployeeID != nil {
		userID = *storedToken.EmployeeID
		// For employees, we'd need to look up their provider_id.
		// For now, we use the employee ID as provider context.
		providerID = *storedToken.EmployeeID
		role = "employee"
	}

	// Generate new tokens
	newAccessToken, err := s.jwtManager.GenerateAccessToken(userID, providerID, role)
	if err != nil {
		return nil, fmt.Errorf("auth.service: %w", err)
	}

	newRefreshToken, err := s.jwtManager.GenerateRefreshToken(userID, providerID, role)
	if err != nil {
		return nil, fmt.Errorf("auth.service: %w", err)
	}

	// Store new refresh token
	newRT := &RefreshToken{
		ProviderID: storedToken.ProviderID,
		EmployeeID: storedToken.EmployeeID,
		TokenHash:  HashToken(newRefreshToken),
		ExpiresAt:  time.Now().Add(s.jwtCfg.RefreshTokenExpiry),
	}
	if err := s.authRepo.StoreRefreshToken(ctx, newRT); err != nil {
		return nil, fmt.Errorf("auth.service: %w", err)
	}

	// Revoke old token (rotation)
	if err := s.authRepo.RevokeToken(ctx, storedToken.ID, &newRT.ID); err != nil {
		return nil, fmt.Errorf("auth.service: %w", err)
	}

	return &TokenPair{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.jwtCfg.AccessTokenExpiry.Seconds()),
	}, nil
}

// Logout revokes a refresh token.
func (s *Service) Logout(ctx context.Context, rawRefreshToken string) error {
	hash := HashToken(rawRefreshToken)

	storedToken, err := s.authRepo.GetByTokenHash(ctx, hash)
	if err != nil {
		// Token not found or already revoked — treat as success
		return nil
	}

	if err := s.authRepo.RevokeToken(ctx, storedToken.ID, nil); err != nil {
		return fmt.Errorf("auth.service: %w", err)
	}

	return nil
}
