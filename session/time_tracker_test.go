package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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

	assert.Equal(t, time.Second*3, elapsed)
}

func newMockedTime(timeValues []time.Time) func() time.Time {
	count := 0

	return func() time.Time {
		val := timeValues[count]
		count = count + 1
		return val
	}
}
