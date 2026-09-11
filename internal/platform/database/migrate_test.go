package database_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/THEcanon001/turnero/internal/platform/database"
)

func TestMigrate_InvalidDSN(t *testing.T) {
	err := database.Migrate("postgres://bad:bad@localhost:1/bad?sslmode=disable")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database.migrate:")
}
