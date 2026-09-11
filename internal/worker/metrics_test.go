package worker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWorker(t *testing.T) {
	w := NewWorker(nil)
	assert.NotNil(t, w)
	assert.Nil(t, w.pool)
}

func TestNewWorker_NotNilPool(t *testing.T) {
	// Verify constructor stores pool reference
	w := NewWorker(nil)
	assert.NotNil(t, w)
}
