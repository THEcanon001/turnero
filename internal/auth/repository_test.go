package auth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/THEcanon001/turnero/internal/auth"
)

func TestHashToken(t *testing.T) {
	token := "my-test-token"

	hash1 := auth.HashToken(token)
	hash2 := auth.HashToken(token)

	assert.Equal(t, hash1, hash2, "same input should produce same hash")
	assert.NotEqual(t, token, hash1, "hash should differ from input")
	assert.Len(t, hash1, 64, "SHA-256 hex should be 64 chars")
}

func TestHashToken_DifferentInputs(t *testing.T) {
	hash1 := auth.HashToken("token-a")
	hash2 := auth.HashToken("token-b")

	assert.NotEqual(t, hash1, hash2, "different inputs should produce different hashes")
}
