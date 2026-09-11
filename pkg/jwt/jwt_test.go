package jwt_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tjwt "github.com/THEcanon001/turnero/pkg/jwt"
)

func newTestManager() *tjwt.Manager {
	return tjwt.NewManager(tjwt.Config{
		Secret:             "test-secret-key-at-least-32-chars!",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "turnero-test",
	})
}

func TestGenerateAccessToken(t *testing.T) {
	m := newTestManager()
	userID := uuid.New()
	providerID := uuid.New()

	token, err := m.GenerateAccessToken(userID, providerID, "admin")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestGenerateRefreshToken(t *testing.T) {
	m := newTestManager()
	userID := uuid.New()
	providerID := uuid.New()

	token, err := m.GenerateRefreshToken(userID, providerID, "employee")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestValidateToken_ValidAccessToken(t *testing.T) {
	m := newTestManager()
	userID := uuid.New()
	providerID := uuid.New()

	token, err := m.GenerateAccessToken(userID, providerID, "admin")
	require.NoError(t, err)

	claims, err := m.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, providerID, claims.ProviderID)
	assert.Equal(t, "admin", claims.Role)
	assert.Equal(t, "access", claims.TokenType)
	assert.Equal(t, "turnero-test", claims.Issuer)
}

func TestValidateToken_ValidRefreshToken(t *testing.T) {
	m := newTestManager()
	userID := uuid.New()
	providerID := uuid.New()

	token, err := m.GenerateRefreshToken(userID, providerID, "employee")
	require.NoError(t, err)

	claims, err := m.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, "employee", claims.Role)
	assert.Equal(t, "refresh", claims.TokenType)
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	m := tjwt.NewManager(tjwt.Config{
		Secret:             "test-secret-key-at-least-32-chars!",
		AccessTokenExpiry:  -1 * time.Hour, // Already expired
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "turnero-test",
	})

	token, err := m.GenerateAccessToken(uuid.New(), uuid.New(), "admin")
	require.NoError(t, err)

	_, err = m.ValidateToken(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestValidateToken_InvalidToken(t *testing.T) {
	m := newTestManager()

	_, err := m.ValidateToken("not-a-valid-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "jwt: invalid token")
}

func TestValidateToken_WrongSecret(t *testing.T) {
	m1 := newTestManager()
	m2 := tjwt.NewManager(tjwt.Config{
		Secret:             "different-secret-key-32-chars!!!",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "turnero-test",
	})

	token, err := m1.GenerateAccessToken(uuid.New(), uuid.New(), "admin")
	require.NoError(t, err)

	_, err = m2.ValidateToken(token)
	assert.Error(t, err)
}

func TestValidateToken_UniqueJTI(t *testing.T) {
	m := newTestManager()
	userID := uuid.New()
	providerID := uuid.New()

	token1, err := m.GenerateAccessToken(userID, providerID, "admin")
	require.NoError(t, err)

	token2, err := m.GenerateAccessToken(userID, providerID, "admin")
	require.NoError(t, err)

	claims1, err := m.ValidateToken(token1)
	require.NoError(t, err)
	claims2, err := m.ValidateToken(token2)
	require.NoError(t, err)

	assert.NotEqual(t, claims1.ID, claims2.ID, "each token should have a unique JTI")
}
