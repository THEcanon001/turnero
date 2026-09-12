package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/THEcanon001/turnero/internal/auth"
	"github.com/THEcanon001/turnero/internal/employee"
	"github.com/THEcanon001/turnero/internal/provider"
	tjwt "github.com/THEcanon001/turnero/pkg/jwt"
	"github.com/THEcanon001/turnero/internal/testutil"
)

var testJWTCfg = tjwt.Config{
	Secret:             "test-secret-key-for-integration-tests",
	AccessTokenExpiry:  15 * time.Minute,
	RefreshTokenExpiry: 7 * 24 * time.Hour,
	Issuer:             "turnero-test",
}

type authDeps struct {
	pool         *pgxpool.Pool
	svc          *auth.Service
	authRepo     *auth.Repository
	providerRepo *provider.Repository
	employeeRepo *employee.Repository
	jwtManager   *tjwt.Manager
	handler      *auth.Handler
	cleanup      func()
}

func setupAuth(t *testing.T) *authDeps {
	t.Helper()
	pool, cleanup := testutil.StartPostgres(t)

	providerRepo := provider.NewRepository(pool)
	employeeRepo := employee.NewRepository(pool)
	authRepo := auth.NewRepository(pool)
	jwtManager := tjwt.NewManager(testJWTCfg)
	svc := auth.NewService(providerRepo, employeeRepo, authRepo, jwtManager, testJWTCfg)
	handler := auth.NewHandler(svc)

	return &authDeps{
		pool: pool, svc: svc, authRepo: authRepo, providerRepo: providerRepo,
		employeeRepo: employeeRepo, jwtManager: jwtManager,
		handler: handler, cleanup: cleanup,
	}
}

func postJSON(handler http.HandlerFunc, url string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

// ===================== Repository Tests =====================

func TestRepository_StoreAndGetRefreshToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	d := setupAuth(t)
	defer d.cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, d.pool, "repo-rt")
	pid := mustUUID(t, providerID)

	rt := &auth.RefreshToken{
		ProviderID: &pid,
		TokenHash:  auth.HashToken("test-refresh-token-1"),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, d.authRepo.StoreRefreshToken(ctx, rt))
	assert.NotEqual(t, uuid.Nil, rt.ID)
	assert.False(t, rt.CreatedAt.IsZero())

	found, err := d.authRepo.GetByTokenHash(ctx, auth.HashToken("test-refresh-token-1"))
	require.NoError(t, err)
	assert.Equal(t, rt.ID, found.ID)
	assert.Equal(t, &pid, found.ProviderID)
	assert.Nil(t, found.RevokedAt)
}

func TestRepository_GetByTokenHash_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	d := setupAuth(t)
	defer d.cleanup()

	_, err := d.authRepo.GetByTokenHash(context.Background(), auth.HashToken("nonexistent"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRepository_RevokeToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	d := setupAuth(t)
	defer d.cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, d.pool, "repo-revoke")
	pid := mustUUID(t, providerID)

	rt := &auth.RefreshToken{
		ProviderID: &pid,
		TokenHash:  auth.HashToken("to-revoke"),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, d.authRepo.StoreRefreshToken(ctx, rt))

	require.NoError(t, d.authRepo.RevokeToken(ctx, rt.ID, nil))

	_, err := d.authRepo.GetByTokenHash(ctx, auth.HashToken("to-revoke"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRepository_RevokeToken_WithReplacement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	d := setupAuth(t)
	defer d.cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, d.pool, "repo-replace")
	pid := mustUUID(t, providerID)

	rt1 := &auth.RefreshToken{ProviderID: &pid, TokenHash: auth.HashToken("original"), ExpiresAt: time.Now().Add(24 * time.Hour)}
	require.NoError(t, d.authRepo.StoreRefreshToken(ctx, rt1))

	rt2 := &auth.RefreshToken{ProviderID: &pid, TokenHash: auth.HashToken("replacement"), ExpiresAt: time.Now().Add(24 * time.Hour)}
	require.NoError(t, d.authRepo.StoreRefreshToken(ctx, rt2))

	require.NoError(t, d.authRepo.RevokeToken(ctx, rt1.ID, &rt2.ID))

	_, err := d.authRepo.GetByTokenHash(ctx, auth.HashToken("original"))
	require.Error(t, err)

	found, err := d.authRepo.GetByTokenHash(ctx, auth.HashToken("replacement"))
	require.NoError(t, err)
	assert.Equal(t, rt2.ID, found.ID)
}

func TestRepository_RevokeAllForProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	d := setupAuth(t)
	defer d.cleanup()
	ctx := context.Background()

	providerID := testutil.SeedProvider(t, d.pool, "repo-revokeall")
	pid := mustUUID(t, providerID)

	for _, tok := range []string{"tok-a", "tok-b"} {
		rt := &auth.RefreshToken{ProviderID: &pid, TokenHash: auth.HashToken(tok), ExpiresAt: time.Now().Add(24 * time.Hour)}
		require.NoError(t, d.authRepo.StoreRefreshToken(ctx, rt))
	}

	require.NoError(t, d.authRepo.RevokeAllForProvider(ctx, pid))

	for _, tok := range []string{"tok-a", "tok-b"} {
		_, err := d.authRepo.GetByTokenHash(ctx, auth.HashToken(tok))
		require.Error(t, err)
	}
}

// ===================== Service Tests =====================

func TestService_Register(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	d := setupAuth(t)
	defer d.cleanup()
	ctx := context.Background()

	t.Run("individual success", func(t *testing.T) {
		resp, err := d.svc.Register(ctx, auth.RegisterRequest{
			Name: "Juan", Email: "juan@test.com", Password: "securepass123",
			Phone: "+5491100000000", Type: "individual", Slug: "juan-peluquero",
		})
		require.NoError(t, err)
		assert.Equal(t, "individual", resp.Type)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)

		claims, err := d.jwtManager.ValidateToken(resp.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, resp.ID, claims.UserID)
		assert.Equal(t, "admin", claims.Role)
	})

	t.Run("business success", func(t *testing.T) {
		bizName := "Barbería"
		resp, err := d.svc.Register(ctx, auth.RegisterRequest{
			Name: "María", Email: "maria@test.com", Password: "securepass123",
			Phone: "+5491100000001", Type: "business", Slug: "barberia",
			BusinessName: &bizName,
		})
		require.NoError(t, err)
		assert.Equal(t, "business", resp.Type)
	})

	t.Run("duplicate email", func(t *testing.T) {
		_, err := d.svc.Register(ctx, auth.RegisterRequest{
			Name: "Dup", Email: "juan@test.com", Password: "securepass123",
			Phone: "+5491100000002", Type: "individual", Slug: "different-slug",
		})
		require.Error(t, err)
	})

	t.Run("duplicate slug", func(t *testing.T) {
		_, err := d.svc.Register(ctx, auth.RegisterRequest{
			Name: "Dup", Email: "diff@test.com", Password: "securepass123",
			Phone: "+5491100000003", Type: "individual", Slug: "juan-peluquero",
		})
		require.Error(t, err)
	})

	t.Run("default timezone", func(t *testing.T) {
		resp, err := d.svc.Register(ctx, auth.RegisterRequest{
			Name: "NoTZ", Email: "notz@test.com", Password: "securepass123",
			Phone: "+5491100000004", Type: "individual", Slug: "no-tz",
		})
		require.NoError(t, err)
		p, err := d.providerRepo.GetByID(ctx, resp.ID)
		require.NoError(t, err)
		assert.Equal(t, "America/Mexico_City", p.Timezone)
	})
}

func TestService_Login(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	d := setupAuth(t)
	defer d.cleanup()
	ctx := context.Background()

	_, err := d.svc.Register(ctx, auth.RegisterRequest{
		Name: "Login User", Email: "login@test.com", Password: "securepass123",
		Phone: "+5491100000005", Type: "individual", Slug: "login-user",
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		resp, err := d.svc.Login(ctx, auth.LoginRequest{Email: "login@test.com", Password: "securepass123"})
		require.NoError(t, err)
		assert.Equal(t, "Bearer", resp.TokenType)
		assert.Equal(t, "Login User", resp.User.Name)
		assert.Equal(t, "admin", resp.User.Role)

		claims, err := d.jwtManager.ValidateToken(resp.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, resp.User.ID, claims.UserID)
	})

	t.Run("wrong password", func(t *testing.T) {
		_, err := d.svc.Login(ctx, auth.LoginRequest{Email: "login@test.com", Password: "wrong"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid credentials")
	})

	t.Run("nonexistent email", func(t *testing.T) {
		_, err := d.svc.Login(ctx, auth.LoginRequest{Email: "noone@test.com", Password: "anything"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid credentials")
	})
}

func TestService_Refresh(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	d := setupAuth(t)
	defer d.cleanup()
	ctx := context.Background()

	regResp, err := d.svc.Register(ctx, auth.RegisterRequest{
		Name: "Refresh", Email: "refresh@test.com", Password: "securepass123",
		Phone: "+5491100000006", Type: "individual", Slug: "refresh-user",
	})
	require.NoError(t, err)

	t.Run("success with rotation", func(t *testing.T) {
		pair, err := d.svc.Refresh(ctx, regResp.RefreshToken)
		require.NoError(t, err)
		assert.NotEmpty(t, pair.AccessToken)
		assert.NotEqual(t, regResp.RefreshToken, pair.RefreshToken)
		assert.Equal(t, "Bearer", pair.TokenType)

		claims, err := d.jwtManager.ValidateToken(pair.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, regResp.ID, claims.UserID)
	})

	t.Run("old token revoked after rotation", func(t *testing.T) {
		_, err := d.svc.Refresh(ctx, regResp.RefreshToken)
		require.Error(t, err)
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := d.svc.Refresh(ctx, "bogus")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid refresh token")
	})
}

func TestService_Logout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	d := setupAuth(t)
	defer d.cleanup()
	ctx := context.Background()

	regResp, err := d.svc.Register(ctx, auth.RegisterRequest{
		Name: "Logout", Email: "logout@test.com", Password: "securepass123",
		Phone: "+5491100000007", Type: "individual", Slug: "logout-user",
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		require.NoError(t, d.svc.Logout(ctx, regResp.RefreshToken))
		_, err := d.svc.Refresh(ctx, regResp.RefreshToken)
		require.Error(t, err)
	})

	t.Run("nonexistent token does not error", func(t *testing.T) {
		require.NoError(t, d.svc.Logout(ctx, "does-not-exist"))
	})
}

func TestService_Join(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	d := setupAuth(t)
	defer d.cleanup()
	ctx := context.Background()

	bizName := "Join Biz"
	regResp, err := d.svc.Register(ctx, auth.RegisterRequest{
		Name: "Owner", Email: "owner@test.com", Password: "securepass123",
		Phone: "+5491100000008", Type: "business", Slug: "join-biz", BusinessName: &bizName,
	})
	require.NoError(t, err)

	t.Run("valid code", func(t *testing.T) {
		inv, err := d.employeeRepo.CreateInvitation(ctx, regResp.ID, "New Emp", 24*time.Hour)
		require.NoError(t, err)

		joinResp, err := d.svc.Join(ctx, auth.JoinRequest{
			Code: inv.Code, Email: "newemp@test.com", Password: "emppass123", Phone: "+5491100000009",
		})
		require.NoError(t, err)
		assert.Equal(t, "New Emp", joinResp.Name)
		assert.Equal(t, regResp.ID, joinResp.ProviderID)
		assert.Equal(t, "employee", joinResp.Role)

		claims, err := d.jwtManager.ValidateToken(joinResp.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, "employee", claims.Role)
	})

	t.Run("invalid code", func(t *testing.T) {
		_, err := d.svc.Join(ctx, auth.JoinRequest{
			Code: "bad", Email: "e@t.com", Password: "pass12345", Phone: "+5491100000010",
		})
		require.Error(t, err)
	})

	t.Run("used code", func(t *testing.T) {
		inv, err := d.employeeRepo.CreateInvitation(ctx, regResp.ID, "Used Emp", 24*time.Hour)
		require.NoError(t, err)

		_, err = d.svc.Join(ctx, auth.JoinRequest{
			Code: inv.Code, Email: "used1@test.com", Password: "pass12345", Phone: "+5491100000011",
		})
		require.NoError(t, err)

		// Try to use same code again
		_, err = d.svc.Join(ctx, auth.JoinRequest{
			Code: inv.Code, Email: "used2@test.com", Password: "pass12345", Phone: "+5491100000012",
		})
		require.Error(t, err)
	})
}

func TestService_LoginEmployee(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	d := setupAuth(t)
	defer d.cleanup()
	ctx := context.Background()

	bizName := "EmpLogin"
	regResp, err := d.svc.Register(ctx, auth.RegisterRequest{
		Name: "Owner", Email: "ownerle@test.com", Password: "securepass123",
		Phone: "+5491100000013", Type: "business", Slug: "emplogin", BusinessName: &bizName,
	})
	require.NoError(t, err)

	inv, err := d.employeeRepo.CreateInvitation(ctx, regResp.ID, "Worker", 24*time.Hour)
	require.NoError(t, err)

	joinResp, err := d.svc.Join(ctx, auth.JoinRequest{
		Code: inv.Code, Email: "worker@test.com", Password: "workerpass", Phone: "+5491100000014",
	})
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		loginResp, err := d.svc.LoginEmployee(ctx, auth.LoginRequest{
			Email: "worker@test.com", Password: "workerpass",
		})
		require.NoError(t, err)
		assert.Equal(t, "Worker", loginResp.User.Name)
		assert.Equal(t, "employee", loginResp.User.Role)
	})

	t.Run("wrong password", func(t *testing.T) {
		_, err := d.svc.LoginEmployee(ctx, auth.LoginRequest{
			Email: "worker@test.com", Password: "wrong",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid credentials")
	})

	t.Run("deactivated", func(t *testing.T) {
		emp, err := d.employeeRepo.GetByID(ctx, joinResp.ID)
		require.NoError(t, err)
		emp.IsActive = false
		require.NoError(t, d.employeeRepo.Update(ctx, emp))

		_, err = d.svc.LoginEmployee(ctx, auth.LoginRequest{
			Email: "worker@test.com", Password: "workerpass",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "deactivated")

		// Re-activate for other tests
		emp.IsActive = true
		require.NoError(t, d.employeeRepo.Update(ctx, emp))
	})
}

func TestService_Refresh_EmployeeToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	d := setupAuth(t)
	defer d.cleanup()
	ctx := context.Background()

	bizName := "RE"
	regResp, err := d.svc.Register(ctx, auth.RegisterRequest{
		Name: "O", Email: "ore@t.com", Password: "securepass123",
		Phone: "+5491100000015", Type: "business", Slug: "re-emp", BusinessName: &bizName,
	})
	require.NoError(t, err)

	inv, err := d.employeeRepo.CreateInvitation(ctx, regResp.ID, "RE", 24*time.Hour)
	require.NoError(t, err)

	joinResp, err := d.svc.Join(ctx, auth.JoinRequest{
		Code: inv.Code, Email: "ere@t.com", Password: "repass1234", Phone: "+5491100000016",
	})
	require.NoError(t, err)

	pair, err := d.svc.Refresh(ctx, joinResp.RefreshToken)
	require.NoError(t, err)

	claims, err := d.jwtManager.ValidateToken(pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, joinResp.ID, claims.UserID)
	assert.Equal(t, "employee", claims.Role)
}

// ===================== Handler HTTP Integration Tests =====================

func TestHandler_Register_HTTP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	d := setupAuth(t)
	defer d.cleanup()

	t.Run("success individual", func(t *testing.T) {
		rec := postJSON(d.handler.Register, "/v1/auth/register", map[string]any{
			"name": "Handler User", "email": "handler@test.com", "password": "securepass123",
			"phone": "+5491100000020", "type": "individual", "slug": "handler-user",
		})
		assert.Equal(t, http.StatusCreated, rec.Code)

		var resp auth.RegisterResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "Handler User", resp.Name)
		assert.NotEmpty(t, resp.AccessToken)
	})

	t.Run("success business", func(t *testing.T) {
		rec := postJSON(d.handler.Register, "/v1/auth/register", map[string]any{
			"name": "Biz", "email": "biz@test.com", "password": "securepass123",
			"phone": "+5491100000021", "type": "business", "slug": "biz-handler",
			"business_name": "My Business",
		})
		assert.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("duplicate email returns 409", func(t *testing.T) {
		rec := postJSON(d.handler.Register, "/v1/auth/register", map[string]any{
			"name": "Dup", "email": "handler@test.com", "password": "securepass123",
			"phone": "+5491100000022", "type": "individual", "slug": "dup-handler",
		})
		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("business missing business_name returns 400", func(t *testing.T) {
		rec := postJSON(d.handler.Register, "/v1/auth/register", map[string]any{
			"name": "NoBiz", "email": "nobiz@test.com", "password": "securepass123",
			"phone": "+5491100000023", "type": "business", "slug": "nobiz",
		})
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHandler_Login_HTTP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	d := setupAuth(t)
	defer d.cleanup()

	// Register first
	postJSON(d.handler.Register, "/v1/auth/register", map[string]any{
		"name": "LoginH", "email": "loginh@test.com", "password": "securepass123",
		"phone": "+5491100000024", "type": "individual", "slug": "loginh",
	})

	t.Run("success", func(t *testing.T) {
		rec := postJSON(d.handler.Login, "/v1/auth/login", map[string]any{
			"email": "loginh@test.com", "password": "securepass123",
		})
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp auth.LoginResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "Bearer", resp.TokenType)
		assert.NotEmpty(t, resp.AccessToken)
		assert.Equal(t, "LoginH", resp.User.Name)
	})

	t.Run("wrong password returns 401", func(t *testing.T) {
		rec := postJSON(d.handler.Login, "/v1/auth/login", map[string]any{
			"email": "loginh@test.com", "password": "wrongpass",
		})
		assert.Equal(t, http.StatusUnauthorized, rec.Code)

		var resp map[string]string
		json.NewDecoder(rec.Body).Decode(&resp)
		assert.Equal(t, "UNAUTHORIZED", resp["code"])
	})

	t.Run("nonexistent email returns 401", func(t *testing.T) {
		rec := postJSON(d.handler.Login, "/v1/auth/login", map[string]any{
			"email": "nobody@test.com", "password": "anything",
		})
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestHandler_Refresh_HTTP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	d := setupAuth(t)
	defer d.cleanup()

	// Register to get tokens
	regRec := postJSON(d.handler.Register, "/v1/auth/register", map[string]any{
		"name": "RefreshH", "email": "refreshh@test.com", "password": "securepass123",
		"phone": "+5491100000025", "type": "individual", "slug": "refreshh",
	})
	var regResp auth.RegisterResponse
	json.NewDecoder(regRec.Body).Decode(&regResp)

	t.Run("success", func(t *testing.T) {
		rec := postJSON(d.handler.Refresh, "/v1/auth/refresh", map[string]any{
			"refresh_token": regResp.RefreshToken,
		})
		assert.Equal(t, http.StatusOK, rec.Code)

		var pair auth.TokenPair
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&pair))
		assert.NotEmpty(t, pair.AccessToken)
		assert.NotEqual(t, regResp.RefreshToken, pair.RefreshToken)
	})

	t.Run("invalid token returns 401", func(t *testing.T) {
		rec := postJSON(d.handler.Refresh, "/v1/auth/refresh", map[string]any{
			"refresh_token": "bogus-token",
		})
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestHandler_Logout_HTTP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	d := setupAuth(t)
	defer d.cleanup()

	regRec := postJSON(d.handler.Register, "/v1/auth/register", map[string]any{
		"name": "LogoutH", "email": "logouth@test.com", "password": "securepass123",
		"phone": "+5491100000026", "type": "individual", "slug": "logouth",
	})
	var regResp auth.RegisterResponse
	json.NewDecoder(regRec.Body).Decode(&regResp)

	t.Run("success returns 204", func(t *testing.T) {
		rec := postJSON(d.handler.Logout, "/v1/auth/logout", map[string]any{
			"refresh_token": regResp.RefreshToken,
		})
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("refresh after logout fails", func(t *testing.T) {
		rec := postJSON(d.handler.Refresh, "/v1/auth/refresh", map[string]any{
			"refresh_token": regResp.RefreshToken,
		})
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestHandler_Join_HTTP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	d := setupAuth(t)
	defer d.cleanup()
	ctx := context.Background()

	// Register business owner
	regRec := postJSON(d.handler.Register, "/v1/auth/register", map[string]any{
		"name": "JoinOwner", "email": "joinowner@test.com", "password": "securepass123",
		"phone": "+5491100000027", "type": "business", "slug": "join-owner",
		"business_name": "JoinBiz",
	})
	var regResp auth.RegisterResponse
	json.NewDecoder(regRec.Body).Decode(&regResp)

	inv, err := d.employeeRepo.CreateInvitation(ctx, regResp.ID, "JoinEmp", 24*time.Hour)
	require.NoError(t, err)

	t.Run("valid code returns 201", func(t *testing.T) {
		rec := postJSON(d.handler.Join, "/v1/auth/join", map[string]any{
			"code": inv.Code, "email": "joinemp@test.com",
			"password": "emppass123", "phone": "+5491100000028",
		})
		assert.Equal(t, http.StatusCreated, rec.Code)

		var joinResp auth.JoinResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&joinResp))
		assert.Equal(t, "JoinEmp", joinResp.Name)
		assert.Equal(t, "employee", joinResp.Role)
	})

	t.Run("invalid code returns 400", func(t *testing.T) {
		rec := postJSON(d.handler.Join, "/v1/auth/join", map[string]any{
			"code": "badcode", "email": "bad@test.com",
			"password": "pass12345", "phone": "+5491100000029",
		})
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var resp map[string]string
		json.NewDecoder(rec.Body).Decode(&resp)
		assert.Equal(t, "INVALID_CODE", resp["code"])
	})
}

func TestHandler_LoginEmployee_HTTP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	d := setupAuth(t)
	defer d.cleanup()
	ctx := context.Background()

	bizName := "LEH"
	regResp, _ := d.svc.Register(ctx, auth.RegisterRequest{
		Name: "Owner", Email: "leh@test.com", Password: "securepass123",
		Phone: "+5491100000030", Type: "business", Slug: "leh-biz", BusinessName: &bizName,
	})

	inv, _ := d.employeeRepo.CreateInvitation(ctx, regResp.ID, "EmpH", 24*time.Hour)
	joinResp, _ := d.svc.Join(ctx, auth.JoinRequest{
		Code: inv.Code, Email: "emph@test.com", Password: "emppass123", Phone: "+5491100000031",
	})

	t.Run("success", func(t *testing.T) {
		rec := postJSON(d.handler.LoginEmployee, "/v1/auth/login/employee", map[string]any{
			"email": "emph@test.com", "password": "emppass123",
		})
		assert.Equal(t, http.StatusOK, rec.Code)

		var resp auth.LoginResponse
		json.NewDecoder(rec.Body).Decode(&resp)
		assert.Equal(t, "EmpH", resp.User.Name)
		assert.Equal(t, "employee", resp.User.Role)
	})

	t.Run("deactivated returns 403", func(t *testing.T) {
		emp, _ := d.employeeRepo.GetByID(ctx, joinResp.ID)
		emp.IsActive = false
		d.employeeRepo.Update(ctx, emp)

		rec := postJSON(d.handler.LoginEmployee, "/v1/auth/login/employee", map[string]any{
			"email": "emph@test.com", "password": "emppass123",
		})
		assert.Equal(t, http.StatusForbidden, rec.Code)

		var resp map[string]string
		json.NewDecoder(rec.Body).Decode(&resp)
		assert.Equal(t, "ACCOUNT_DEACTIVATED", resp["code"])
	})
}

// Helpers

func mustUUID(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	require.NoError(t, err)
	return id
}

