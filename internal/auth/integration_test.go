package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/THEcanon001/turnero/internal/auth"
	"github.com/THEcanon001/turnero/internal/employee"
	"github.com/THEcanon001/turnero/internal/provider"
	tjwt "github.com/THEcanon001/turnero/pkg/jwt"
	"github.com/THEcanon001/turnero/internal/testutil"
)

var jwtCfg = tjwt.Config{
	Secret:             "test-secret-key-for-integration-tests",
	AccessTokenExpiry:  15 * time.Minute,
	RefreshTokenExpiry: 7 * 24 * time.Hour,
	Issuer:             "turnero-test",
}

// setupAuthService creates all dependencies for the auth service using a real DB.
func setupAuthService(t *testing.T) (*auth.Service, *auth.Repository, *provider.Repository, *employee.Repository, *tjwt.Manager, func()) {
	t.Helper()
	pool, cleanup := testutil.StartPostgres(t)

	providerRepo := provider.NewRepository(pool)
	employeeRepo := employee.NewRepository(pool)
	authRepo := auth.NewRepository(pool)
	jwtManager := tjwt.NewManager(jwtCfg)

	svc := auth.NewService(providerRepo, employeeRepo, authRepo, jwtManager, jwtCfg)

	return svc, authRepo, providerRepo, employeeRepo, jwtManager, cleanup
}

// ---------- Repository tests ----------

func TestRepository_StoreAndGetRefreshToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := auth.NewRepository(pool)
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "repo-rt")
	pid, err := uuid.Parse(providerID)
	require.NoError(t, err)

	rt := &auth.RefreshToken{
		ProviderID: &pid,
		TokenHash:  auth.HashToken("test-refresh-token-1"),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}

	err = repo.StoreRefreshToken(ctx, rt)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, rt.ID)
	assert.False(t, rt.CreatedAt.IsZero())

	// Retrieve by hash
	found, err := repo.GetByTokenHash(ctx, auth.HashToken("test-refresh-token-1"))
	require.NoError(t, err)
	assert.Equal(t, rt.ID, found.ID)
	assert.Equal(t, &pid, found.ProviderID)
	assert.Nil(t, found.RevokedAt)
}

func TestRepository_GetByTokenHash_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := auth.NewRepository(pool)
	_, err := repo.GetByTokenHash(context.Background(), auth.HashToken("nonexistent"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRepository_RevokeToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := auth.NewRepository(pool)
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "repo-revoke")
	pid, err := uuid.Parse(providerID)
	require.NoError(t, err)

	rt := &auth.RefreshToken{
		ProviderID: &pid,
		TokenHash:  auth.HashToken("to-revoke"),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, repo.StoreRefreshToken(ctx, rt))

	err = repo.RevokeToken(ctx, rt.ID, nil)
	require.NoError(t, err)

	// Revoked tokens are excluded from GetByTokenHash
	_, err = repo.GetByTokenHash(ctx, auth.HashToken("to-revoke"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRepository_RevokeToken_WithReplacement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := auth.NewRepository(pool)
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "repo-replace")
	pid, err := uuid.Parse(providerID)
	require.NoError(t, err)

	// Original token
	rt1 := &auth.RefreshToken{
		ProviderID: &pid,
		TokenHash:  auth.HashToken("original"),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, repo.StoreRefreshToken(ctx, rt1))

	// Replacement token
	rt2 := &auth.RefreshToken{
		ProviderID: &pid,
		TokenHash:  auth.HashToken("replacement"),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, repo.StoreRefreshToken(ctx, rt2))

	// Revoke original linking to replacement
	err = repo.RevokeToken(ctx, rt1.ID, &rt2.ID)
	require.NoError(t, err)

	// Original gone, replacement still valid
	_, err = repo.GetByTokenHash(ctx, auth.HashToken("original"))
	require.Error(t, err)

	found, err := repo.GetByTokenHash(ctx, auth.HashToken("replacement"))
	require.NoError(t, err)
	assert.Equal(t, rt2.ID, found.ID)
}

func TestRepository_RevokeAllForProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := testutil.StartPostgres(t)
	defer cleanup()

	repo := auth.NewRepository(pool)
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, pool, "repo-revokeall")
	pid, err := uuid.Parse(providerID)
	require.NoError(t, err)

	for _, tok := range []string{"tok-a", "tok-b"} {
		rt := &auth.RefreshToken{
			ProviderID: &pid,
			TokenHash:  auth.HashToken(tok),
			ExpiresAt:  time.Now().Add(24 * time.Hour),
		}
		require.NoError(t, repo.StoreRefreshToken(ctx, rt))
	}

	err = repo.RevokeAllForProvider(ctx, pid)
	require.NoError(t, err)

	_, err = repo.GetByTokenHash(ctx, auth.HashToken("tok-a"))
	require.Error(t, err)
	_, err = repo.GetByTokenHash(ctx, auth.HashToken("tok-b"))
	require.Error(t, err)
}

// ---------- Service tests ----------

func TestService_Register_Individual(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	svc, _, providerRepo, _, jwtManager, cleanup := setupAuthService(t)
	defer cleanup()
	ctx := context.Background()

	resp, err := svc.Register(ctx, auth.RegisterRequest{
		Name:     "Juan Peluquero",
		Email:    "juan@test.com",
		Password: "securepass123",
		Phone:    "+5491100000000",
		Type:     "individual",
		Slug:     "juan-peluquero",
	})

	require.NoError(t, err)
	assert.Equal(t, "Juan Peluquero", resp.Name)
	assert.Equal(t, "juan@test.com", resp.Email)
	assert.Equal(t, "individual", resp.Type)
	assert.Equal(t, "juan-peluquero", resp.Slug)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)

	// Verify provider in DB
	p, err := providerRepo.GetByEmail(ctx, "juan@test.com")
	require.NoError(t, err)
	assert.Equal(t, resp.ID, p.ID)

	// Verify access token
	claims, err := jwtManager.ValidateToken(resp.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, resp.ID, claims.UserID)
	assert.Equal(t, "admin", claims.Role)
	assert.Equal(t, "access", claims.TokenType)
}

func TestService_Register_Business(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	svc, _, _, _, _, cleanup := setupAuthService(t)
	defer cleanup()

	bizName := "Barbería Moderna"
	resp, err := svc.Register(context.Background(), auth.RegisterRequest{
		Name:         "María García",
		Email:        "maria@test.com",
		Password:     "securepass123",
		Phone:        "+5491100000001",
		Type:         "business",
		Slug:         "barberia-moderna",
		BusinessName: &bizName,
	})

	require.NoError(t, err)
	assert.Equal(t, "business", resp.Type)
	assert.NotEmpty(t, resp.AccessToken)
}

func TestService_Register_DefaultTimezone(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	svc, _, providerRepo, _, _, cleanup := setupAuthService(t)
	defer cleanup()
	ctx := context.Background()

	resp, err := svc.Register(ctx, auth.RegisterRequest{
		Name:     "No TZ",
		Email:    "notz@test.com",
		Password: "securepass123",
		Phone:    "+5491100000099",
		Type:     "individual",
		Slug:     "no-tz",
	})
	require.NoError(t, err)

	p, err := providerRepo.GetByID(ctx, resp.ID)
	require.NoError(t, err)
	assert.Equal(t, "America/Mexico_City", p.Timezone)
}

func TestService_Register_DuplicateSlug(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	svc, _, _, _, _, cleanup := setupAuthService(t)
	defer cleanup()
	ctx := context.Background()

	_, err := svc.Register(ctx, auth.RegisterRequest{
		Name: "First", Email: "first@test.com", Password: "securepass123",
		Phone: "+5491100000002", Type: "individual", Slug: "same-slug",
	})
	require.NoError(t, err)

	_, err = svc.Register(ctx, auth.RegisterRequest{
		Name: "Second", Email: "second@test.com", Password: "securepass123",
		Phone: "+5491100000003", Type: "individual", Slug: "same-slug",
	})
	require.Error(t, err)
}

func TestService_Login_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	svc, _, _, _, jwtManager, cleanup := setupAuthService(t)
	defer cleanup()
	ctx := context.Background()

	_, err := svc.Register(ctx, auth.RegisterRequest{
		Name: "Login User", Email: "login@test.com", Password: "securepass123",
		Phone: "+5491100000003", Type: "individual", Slug: "login-user",
	})
	require.NoError(t, err)

	resp, err := svc.Login(ctx, auth.LoginRequest{
		Email: "login@test.com", Password: "securepass123",
	})

	require.NoError(t, err)
	assert.Equal(t, "Bearer", resp.TokenType)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "Login User", resp.User.Name)
	assert.Equal(t, "admin", resp.User.Role)

	claims, err := jwtManager.ValidateToken(resp.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, resp.User.ID, claims.UserID)
}

func TestService_Login_WrongPassword(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	svc, _, _, _, _, cleanup := setupAuthService(t)
	defer cleanup()
	ctx := context.Background()

	_, err := svc.Register(ctx, auth.RegisterRequest{
		Name: "Wrong", Email: "wrong@test.com", Password: "securepass123",
		Phone: "+5491100000004", Type: "individual", Slug: "wrong-pass",
	})
	require.NoError(t, err)

	_, err = svc.Login(ctx, auth.LoginRequest{
		Email: "wrong@test.com", Password: "wrongpassword",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestService_Login_NonexistentEmail(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	svc, _, _, _, _, cleanup := setupAuthService(t)
	defer cleanup()

	_, err := svc.Login(context.Background(), auth.LoginRequest{
		Email: "noone@test.com", Password: "anything",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestService_Refresh_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	svc, _, _, _, jwtManager, cleanup := setupAuthService(t)
	defer cleanup()
	ctx := context.Background()

	regResp, err := svc.Register(ctx, auth.RegisterRequest{
		Name: "Refresh", Email: "refresh@test.com", Password: "securepass123",
		Phone: "+5491100000005", Type: "individual", Slug: "refresh-user",
	})
	require.NoError(t, err)

	pair, err := svc.Refresh(ctx, regResp.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.Equal(t, "Bearer", pair.TokenType)
	assert.NotEqual(t, regResp.RefreshToken, pair.RefreshToken) // rotated

	claims, err := jwtManager.ValidateToken(pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, regResp.ID, claims.UserID)

	// Old token revoked
	_, err = svc.Refresh(ctx, regResp.RefreshToken)
	require.Error(t, err)
}

func TestService_Refresh_InvalidToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	svc, _, _, _, _, cleanup := setupAuthService(t)
	defer cleanup()

	_, err := svc.Refresh(context.Background(), "bogus")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid refresh token")
}

func TestService_Logout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	svc, _, _, _, _, cleanup := setupAuthService(t)
	defer cleanup()
	ctx := context.Background()

	regResp, err := svc.Register(ctx, auth.RegisterRequest{
		Name: "Logout", Email: "logout@test.com", Password: "securepass123",
		Phone: "+5491100000006", Type: "individual", Slug: "logout-user",
	})
	require.NoError(t, err)

	require.NoError(t, svc.Logout(ctx, regResp.RefreshToken))

	// Refresh should fail
	_, err = svc.Refresh(ctx, regResp.RefreshToken)
	require.Error(t, err)
}

func TestService_Logout_NonexistentToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	svc, _, _, _, _, cleanup := setupAuthService(t)
	defer cleanup()

	err := svc.Logout(context.Background(), "does-not-exist")
	require.NoError(t, err)
}

func TestService_Join_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	svc, _, _, employeeRepo, jwtManager, cleanup := setupAuthService(t)
	defer cleanup()
	ctx := context.Background()

	bizName := "Join Biz"
	regResp, err := svc.Register(ctx, auth.RegisterRequest{
		Name: "Owner", Email: "owner-join@test.com", Password: "securepass123",
		Phone: "+5491100000007", Type: "business", Slug: "join-biz", BusinessName: &bizName,
	})
	require.NoError(t, err)

	inv, err := employeeRepo.CreateInvitation(ctx, regResp.ID, "New Emp", 24*time.Hour)
	require.NoError(t, err)

	joinResp, err := svc.Join(ctx, auth.JoinRequest{
		Code: inv.Code, Email: "newemp@test.com", Password: "emppass123", Phone: "+5491100000008",
	})
	require.NoError(t, err)
	assert.Equal(t, "New Emp", joinResp.Name)
	assert.Equal(t, regResp.ID, joinResp.ProviderID)
	assert.Equal(t, "employee", joinResp.Role)

	claims, err := jwtManager.ValidateToken(joinResp.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, joinResp.ID, claims.UserID)
	assert.Equal(t, "employee", claims.Role)
}

func TestService_Join_InvalidCode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	svc, _, _, _, _, cleanup := setupAuthService(t)
	defer cleanup()

	_, err := svc.Join(context.Background(), auth.JoinRequest{
		Code: "bad", Email: "e@t.com", Password: "pass12345", Phone: "+5491100000009",
	})
	require.Error(t, err)
}

func TestService_LoginEmployee_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	svc, _, _, employeeRepo, jwtManager, cleanup := setupAuthService(t)
	defer cleanup()
	ctx := context.Background()

	bizName := "EmpLogin"
	regResp, err := svc.Register(ctx, auth.RegisterRequest{
		Name: "Owner", Email: "ownerle@test.com", Password: "securepass123",
		Phone: "+5491100000010", Type: "business", Slug: "emplogin", BusinessName: &bizName,
	})
	require.NoError(t, err)

	inv, err := employeeRepo.CreateInvitation(ctx, regResp.ID, "Worker", 24*time.Hour)
	require.NoError(t, err)

	_, err = svc.Join(ctx, auth.JoinRequest{
		Code: inv.Code, Email: "worker@test.com", Password: "workerpass", Phone: "+5491100000011",
	})
	require.NoError(t, err)

	loginResp, err := svc.LoginEmployee(ctx, auth.LoginRequest{
		Email: "worker@test.com", Password: "workerpass",
	})
	require.NoError(t, err)
	assert.Equal(t, "Worker", loginResp.User.Name)
	assert.Equal(t, "employee", loginResp.User.Role)

	claims, err := jwtManager.ValidateToken(loginResp.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, loginResp.User.ID, claims.UserID)
}

func TestService_LoginEmployee_WrongPassword(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	svc, _, _, employeeRepo, _, cleanup := setupAuthService(t)
	defer cleanup()
	ctx := context.Background()

	bizName := "WP"
	regResp, err := svc.Register(ctx, auth.RegisterRequest{
		Name: "O", Email: "owp@t.com", Password: "securepass123",
		Phone: "+5491100000012", Type: "business", Slug: "wp-emp", BusinessName: &bizName,
	})
	require.NoError(t, err)

	inv, err := employeeRepo.CreateInvitation(ctx, regResp.ID, "E", 24*time.Hour)
	require.NoError(t, err)

	_, err = svc.Join(ctx, auth.JoinRequest{
		Code: inv.Code, Email: "ewp@t.com", Password: "correct123", Phone: "+5491100000013",
	})
	require.NoError(t, err)

	_, err = svc.LoginEmployee(ctx, auth.LoginRequest{
		Email: "ewp@t.com", Password: "wrong",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestService_LoginEmployee_Deactivated(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	svc, _, _, employeeRepo, _, cleanup := setupAuthService(t)
	defer cleanup()
	ctx := context.Background()

	bizName := "DA"
	regResp, err := svc.Register(ctx, auth.RegisterRequest{
		Name: "O", Email: "oda@t.com", Password: "securepass123",
		Phone: "+5491100000014", Type: "business", Slug: "da-emp", BusinessName: &bizName,
	})
	require.NoError(t, err)

	inv, err := employeeRepo.CreateInvitation(ctx, regResp.ID, "DA", 24*time.Hour)
	require.NoError(t, err)

	joinResp, err := svc.Join(ctx, auth.JoinRequest{
		Code: inv.Code, Email: "eda@t.com", Password: "dapass1234", Phone: "+5491100000015",
	})
	require.NoError(t, err)

	emp, err := employeeRepo.GetByID(ctx, joinResp.ID)
	require.NoError(t, err)
	emp.IsActive = false
	require.NoError(t, employeeRepo.Update(ctx, emp))

	_, err = svc.LoginEmployee(ctx, auth.LoginRequest{
		Email: "eda@t.com", Password: "dapass1234",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deactivated")
}

func TestService_Refresh_EmployeeToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	svc, _, _, employeeRepo, jwtManager, cleanup := setupAuthService(t)
	defer cleanup()
	ctx := context.Background()

	bizName := "RE"
	regResp, err := svc.Register(ctx, auth.RegisterRequest{
		Name: "O", Email: "ore@t.com", Password: "securepass123",
		Phone: "+5491100000016", Type: "business", Slug: "re-emp", BusinessName: &bizName,
	})
	require.NoError(t, err)

	inv, err := employeeRepo.CreateInvitation(ctx, regResp.ID, "RE", 24*time.Hour)
	require.NoError(t, err)

	joinResp, err := svc.Join(ctx, auth.JoinRequest{
		Code: inv.Code, Email: "ere@t.com", Password: "repass1234", Phone: "+5491100000017",
	})
	require.NoError(t, err)

	pair, err := svc.Refresh(ctx, joinResp.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)

	claims, err := jwtManager.ValidateToken(pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, joinResp.ID, claims.UserID)
	assert.Equal(t, "employee", claims.Role)
}
