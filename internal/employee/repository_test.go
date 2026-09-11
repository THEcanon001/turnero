package employee

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCode(t *testing.T) {
	code, err := generateCode(8)
	require.NoError(t, err)
	assert.Len(t, code, 8)

	// Verify codes are unique
	code2, err := generateCode(8)
	require.NoError(t, err)
	assert.NotEqual(t, code, code2)
}

func TestGenerateCode_DifferentLengths(t *testing.T) {
	tests := []int{4, 6, 8, 12, 16}
	for _, length := range tests {
		code, err := generateCode(length)
		require.NoError(t, err)
		assert.Len(t, code, length)
	}
}
