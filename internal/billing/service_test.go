package billing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewService(t *testing.T) {
	s := NewService(nil)
	assert.NotNil(t, s)
	assert.Nil(t, s.repo)
}

func TestNewService_WithRepo(t *testing.T) {
	repo := &Repository{}
	s := NewService(repo)
	assert.NotNil(t, s)
	assert.Equal(t, repo, s.repo)
}
