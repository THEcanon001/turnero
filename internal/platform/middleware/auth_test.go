package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/THEcanon001/turnero/internal/platform/middleware"
	tjwt "github.com/THEcanon001/turnero/pkg/jwt"
)

func newTestJWTManager() *tjwt.Manager {
	return tjwt.NewManager(tjwt.Config{
		Secret:             "test-secret-key-at-least-32-chars!",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "turnero-test",
	})
}

func TestAuth_ValidToken(t *testing.T) {
	jm := newTestJWTManager()
	userID := uuid.New()
	providerID := uuid.New()

	token, err := jm.GenerateAccessToken(userID, providerID, "admin")
	require.NoError(t, err)

	var capturedUserID uuid.UUID
	var capturedRole string

	handler := middleware.Auth(jm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = middleware.GetUserID(r.Context())
		capturedRole = middleware.GetRole(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, userID, capturedUserID)
	assert.Equal(t, "admin", capturedRole)
}

func TestAuth_MissingHeader(t *testing.T) {
	jm := newTestJWTManager()
	handler := middleware.Auth(jm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_InvalidFormat(t *testing.T) {
	jm := newTestJWTManager()
	handler := middleware.Auth(jm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "NotBearer token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_InvalidToken(t *testing.T) {
	jm := newTestJWTManager()
	handler := middleware.Auth(jm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_ExpiredToken(t *testing.T) {
	jm := tjwt.NewManager(tjwt.Config{
		Secret:             "test-secret-key-at-least-32-chars!",
		AccessTokenExpiry:  -1 * time.Hour,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "turnero-test",
	})

	token, err := jm.GenerateAccessToken(uuid.New(), uuid.New(), "admin")
	require.NoError(t, err)

	handler := middleware.Auth(jm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "TOKEN_EXPIRED")
}

func TestAuth_RefreshTokenRejected(t *testing.T) {
	jm := newTestJWTManager()
	// Generate a refresh token, not an access token
	token, err := jm.GenerateRefreshToken(uuid.New(), uuid.New(), "admin")
	require.NoError(t, err)

	handler := middleware.Auth(jm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireRole_Allowed(t *testing.T) {
	jm := newTestJWTManager()
	token, _ := jm.GenerateAccessToken(uuid.New(), uuid.New(), "admin")

	handler := middleware.Auth(jm)(
		middleware.RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireRole_Forbidden(t *testing.T) {
	jm := newTestJWTManager()
	token, _ := jm.GenerateAccessToken(uuid.New(), uuid.New(), "employee")

	handler := middleware.Auth(jm)(
		middleware.RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		})),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestGetUserID_NoAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	id := middleware.GetUserID(req.Context())
	assert.Equal(t, uuid.Nil, id)
}

func TestGetProviderID_NoAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	id := middleware.GetProviderID(req.Context())
	assert.Equal(t, uuid.Nil, id)
}

func TestGetRole_NoAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	role := middleware.GetRole(req.Context())
	assert.Empty(t, role)
}
