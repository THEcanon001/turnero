package notification

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsInvalidTokenError(t *testing.T) {
	assert.True(t, isInvalidTokenError(fmt.Errorf("NotRegistered")))
	assert.True(t, isInvalidTokenError(fmt.Errorf("InvalidRegistration")))
	assert.True(t, isInvalidTokenError(fmt.Errorf("FCM returned status 401")))
	assert.False(t, isInvalidTokenError(nil))
	assert.False(t, isInvalidTokenError(fmt.Errorf("timeout")))
}

func TestNewService(t *testing.T) {
	s := NewService(nil, "", "")
	assert.NotNil(t, s)
	assert.Equal(t, "https://fcm.googleapis.com/fcm/send", s.fcmURL)
}

func TestNewService_CustomURL(t *testing.T) {
	s := NewService(nil, "http://custom:8080", "key123")
	assert.Equal(t, "http://custom:8080", s.fcmURL)
	assert.Equal(t, "key123", s.fcmAPIKey)
}
