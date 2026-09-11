package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/THEcanon001/turnero/internal/employee"
	"github.com/THEcanon001/turnero/internal/provider"
	tjwt "github.com/THEcanon001/turnero/pkg/jwt"
)

const bcryptCost = 12

// Service handles authentication business logic.
type Service struct {
	providerRepo *provider.Repository
	employeeRepo *employee.Repository
	authRepo     *Repository
	jwtManager   *tjwt.Manager
	jwtCfg       tjwt.Config
}

// NewService creates a new auth service.
func NewService(
	providerRepo *provider.Repository,
	employeeRepo *employee.Repository,
	authRepo *Repository,
	jwtManager *tjwt.Manager,
	jwtCfg tjwt.Config,
) *Service {
	return &Service{
		providerRepo: providerRepo,
		employeeRepo: employeeRepo,
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

	// Auto-create "self" employee for individual (PF) providers
	if p.Type == provider.TypeIndividual {
		selfEmployee := &employee.Employee{
			ProviderID: p.ID,
			Name:       p.Name,
			Phone:      p.Phone,
			Role:       employee.RoleAdmin,
			Email:      &p.Email,
		}
		if err := s.employeeRepo.Create(ctx, selfEmployee); err != nil {
			return nil, fmt.Errorf("auth.service: create self employee: %w", err)
		}
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
		emp, err := s.employeeRepo.GetByID(ctx, *storedToken.EmployeeID)
		if err != nil {
			return nil, fmt.Errorf("auth.service: employee not found")
		}
		providerID = emp.ProviderID
		role = string(emp.Role)
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

// Join allows an employee to accept an invitation code and create their account.
func (s *Service) Join(ctx context.Context, req JoinRequest) (*JoinResponse, error) {
	inv, err := s.employeeRepo.GetInvitationByCode(ctx, req.Code)
	if err != nil {
		return nil, fmt.Errorf("auth.service: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("auth.service: hash password: %w", err)
	}

	hashStr := string(hash)
	emp := &employee.Employee{
		ProviderID:   inv.ProviderID,
		Name:         inv.EmployeeName,
		Phone:        req.Phone,
		Role:         employee.RoleEmployee,
		Email:        &req.Email,
		PasswordHash: &hashStr,
	}

	if err := s.employeeRepo.Create(ctx, emp); err != nil {
		return nil, fmt.Errorf("auth.service: create employee: %w", err)
	}

	if err := s.employeeRepo.MarkInvitationUsed(ctx, inv.ID, emp.ID); err != nil {
		return nil, fmt.Errorf("auth.service: %w", err)
	}

	accessToken, err := s.jwtManager.GenerateAccessToken(emp.ID, inv.ProviderID, string(emp.Role))
	if err != nil {
		return nil, fmt.Errorf("auth.service: %w", err)
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(emp.ID, inv.ProviderID, string(emp.Role))
	if err != nil {
		return nil, fmt.Errorf("auth.service: %w", err)
	}

	rt := &RefreshToken{
		EmployeeID: &emp.ID,
		TokenHash:  HashToken(refreshToken),
		ExpiresAt:  time.Now().Add(s.jwtCfg.RefreshTokenExpiry),
	}
	if err := s.authRepo.StoreRefreshToken(ctx, rt); err != nil {
		return nil, fmt.Errorf("auth.service: %w", err)
	}

	return &JoinResponse{
		ID:           emp.ID,
		Name:         emp.Name,
		Email:        req.Email,
		ProviderID:   inv.ProviderID,
		Role:         string(emp.Role),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// LoginEmployee authenticates an employee by email/password.
func (s *Service) LoginEmployee(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	emp, err := s.employeeRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("auth.service: invalid credentials")
	}

	if emp.PasswordHash == nil {
		return nil, fmt.Errorf("auth.service: invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*emp.PasswordHash), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("auth.service: invalid credentials")
	}

	if !emp.IsActive {
		return nil, fmt.Errorf("auth.service: account deactivated")
	}

	accessToken, err := s.jwtManager.GenerateAccessToken(emp.ID, emp.ProviderID, string(emp.Role))
	if err != nil {
		return nil, fmt.Errorf("auth.service: %w", err)
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(emp.ID, emp.ProviderID, string(emp.Role))
	if err != nil {
		return nil, fmt.Errorf("auth.service: %w", err)
	}

	rt := &RefreshToken{
		EmployeeID: &emp.ID,
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
			ID:         emp.ID,
			Name:       emp.Name,
			Role:       string(emp.Role),
			Type:       "employee",
			ProviderID: emp.ProviderID,
		},
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
