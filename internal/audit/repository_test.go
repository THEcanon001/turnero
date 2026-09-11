package audit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRepository(t *testing.T) {
	r := NewRepository(nil)
	assert.NotNil(t, r)
	assert.Nil(t, r.pool)
}
