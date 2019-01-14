package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_NotStartedTrackerElapsedReturnsZeroValue(t *testing.T) {
	tt := NewTracker(func() time.Time { return time.Unix(10, 10) })

	assert.Equal(t, 0*time.Second, tt.Elapsed())
}

func Test_ElapsedReturnsCorrectValue(t *testing.T) {
	mockedClock := newMockedTime(
		[]time.Time{
			time.Unix(1, 0),
			time.Unix(4, 0),
		},
	)
	tt := NewTracker(mockedClock)

	tt.StartTracking()
	elapsed := tt.Elapsed()

	assert.Equal(t, 3*time.Second, elapsed)
}

func newMockedTime(timeValues []time.Time) func() time.Time {
	count := 0

	return func() time.Time {
		val := timeValues[count]
		count = count + 1
		return val
	}
}
