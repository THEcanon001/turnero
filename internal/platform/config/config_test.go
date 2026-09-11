package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/THEcanon001/turnero/internal/platform/config"
)

func TestLoad_WithRequiredEnvVars(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "test-secret-key", cfg.JWT.Secret)
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	cfg, err := config.Load()
	require.NoError(t, err)

	// Server defaults
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)

	// Database defaults
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "turnero", cfg.Database.User)
	assert.Equal(t, "turnero", cfg.Database.Name)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
	assert.Equal(t, 25, cfg.Database.MaxOpenConns)
	assert.Equal(t, 5, cfg.Database.MaxIdleConns)

	// JWT defaults
	assert.Equal(t, "turnero", cfg.JWT.Issuer)
}

func TestLoad_MissingRequiredJWTSecret(t *testing.T) {
	// Unset JWT_SECRET to trigger required field error
	t.Setenv("JWT_SECRET", "")

	_, err := config.Load()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config:")
}

func TestLoad_CustomValues(t *testing.T) {
	t.Setenv("JWT_SECRET", "custom-secret")
	t.Setenv("SERVER_HOST", "127.0.0.1")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("DB_HOST", "db.example.com")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_USER", "admin")
	t.Setenv("DB_PASSWORD", "s3cret")
	t.Setenv("DB_NAME", "mydb")
	t.Setenv("DB_SSL_MODE", "require")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "db.example.com", cfg.Database.Host)
	assert.Equal(t, 5433, cfg.Database.Port)
	assert.Equal(t, "admin", cfg.Database.User)
	assert.Equal(t, "s3cret", cfg.Database.Password)
	assert.Equal(t, "mydb", cfg.Database.Name)
	assert.Equal(t, "require", cfg.Database.SSLMode)
}

func TestServerConfig_Addr(t *testing.T) {
	cfg := config.ServerConfig{Host: "0.0.0.0", Port: 8080}
	assert.Equal(t, "0.0.0.0:8080", cfg.Addr())
}

func TestDatabaseConfig_DSN(t *testing.T) {
	cfg := config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "user",
		Password: "pass",
		Name:     "db",
		SSLMode:  "disable",
	}
	expected := "postgres://user:pass@localhost:5432/db?sslmode=disable"
	assert.Equal(t, expected, cfg.DSN())
}
