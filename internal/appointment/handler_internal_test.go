package appointment

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCheckCancelDeadline_FarFuture(t *testing.T) {
	future := time.Now().Add(24 * time.Hour).Format("2006-01-02")
	err := checkCancelDeadline(future, "10:00")
	assert.NoError(t, err)
}

func TestCheckCancelDeadline_TooClose(t *testing.T) {
	soon := time.Now().Add(30 * time.Minute).Format("2006-01-02")
	soonTime := time.Now().Add(30 * time.Minute).Format("15:04")
	err := checkCancelDeadline(soon, soonTime)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot cancel within")
}

func TestCheckCancelDeadline_InvalidDate(t *testing.T) {
	err := checkCancelDeadline("not-a-date", "10:00")
	assert.NoError(t, err) // invalid dates don't block cancellation
}
