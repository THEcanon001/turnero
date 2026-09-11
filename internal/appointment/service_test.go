package appointment

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTime_HHMM(t *testing.T) {
	result, err := parseTime("09:30")
	require.NoError(t, err)
	assert.Equal(t, 9, result.Hour())
	assert.Equal(t, 30, result.Minute())
}

func TestParseTime_HHMMSS(t *testing.T) {
	result, err := parseTime("14:00:00")
	require.NoError(t, err)
	assert.Equal(t, 14, result.Hour())
	assert.Equal(t, 0, result.Minute())
}

func TestParseTime_Invalid(t *testing.T) {
	_, err := parseTime("bad")
	assert.Error(t, err)
}

func TestSlotGeneration(t *testing.T) {
	// Simulate slot generation logic for 09:00-12:00 with 30min slots, 0 break
	startTime, _ := time.Parse("15:04", "09:00")
	endTime, _ := time.Parse("15:04", "12:00")
	slotDuration := 30 * time.Minute
	breakDuration := 0 * time.Minute

	booked := map[string]bool{
		"10:00": true,
	}

	var slots []Slot
	current := startTime
	for current.Add(slotDuration).Before(endTime) || current.Add(slotDuration).Equal(endTime) {
		slotStart := current.Format("15:04")
		slotEnd := current.Add(slotDuration).Format("15:04")

		slots = append(slots, Slot{
			StartTime: slotStart,
			EndTime:   slotEnd,
			Available: !booked[slotStart],
		})

		current = current.Add(slotDuration + breakDuration)
	}

	assert.Len(t, slots, 6)
	assert.True(t, slots[0].Available)  // 09:00
	assert.True(t, slots[1].Available)  // 09:30
	assert.False(t, slots[2].Available) // 10:00 (booked)
	assert.True(t, slots[3].Available)  // 10:30
	assert.True(t, slots[4].Available)  // 11:00
	assert.True(t, slots[5].Available)  // 11:30
}

func TestSlotGeneration_WithBreak(t *testing.T) {
	startTime, _ := time.Parse("15:04", "09:00")
	endTime, _ := time.Parse("15:04", "11:00")
	slotDuration := 30 * time.Minute
	breakDuration := 10 * time.Minute

	var slots []Slot
	current := startTime
	for current.Add(slotDuration).Before(endTime) || current.Add(slotDuration).Equal(endTime) {
		slotStart := current.Format("15:04")
		slotEnd := current.Add(slotDuration).Format("15:04")

		slots = append(slots, Slot{
			StartTime: slotStart,
			EndTime:   slotEnd,
			Available: true,
		})

		current = current.Add(slotDuration + breakDuration)
	}

	// 09:00-09:30 (next at 09:40), 09:40-10:10 (next at 10:20), 10:20-10:50 (next at 11:00 but 11:00+30 > 11:00)
	assert.Len(t, slots, 3)
	assert.Equal(t, "09:00", slots[0].StartTime)
	assert.Equal(t, "09:40", slots[1].StartTime)
	assert.Equal(t, "10:20", slots[2].StartTime)
}
