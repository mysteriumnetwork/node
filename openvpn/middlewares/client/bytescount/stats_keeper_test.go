package bytescount

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type fakeClock struct {
	time time.Time
}

func (fc *fakeClock) SetTime(time time.Time) {
	fc.time = time
}

func (fc *fakeClock) GetTime() time.Time {
	return fc.time
}

func TestStatsSavingWorks(t *testing.T) {
	statsKeeper := NewSessionStatsKeeper(time.Now)
	stats := SessionStats{BytesSent: 1, BytesReceived: 2}

	statsKeeper.Save(stats)
	assert.Equal(t, stats, statsKeeper.Retrieve())
}

func TestGetSessionDurationReturnsFlooredDuration(t *testing.T) {
	clock := fakeClock{}
	statsKeeper := NewSessionStatsKeeper(clock.GetTime)

	clock.SetTime(time.Date(2000, time.January, 0, 10, 12, 3, 0, time.UTC))
	statsKeeper.MarkSessionStart()

	clock.SetTime(time.Date(2000, time.January, 0, 10, 12, 4, 700000000, time.UTC))
	expectedDuration, err := time.ParseDuration("1s700000000ns")
	if err != nil {
		assert.FailNow(t, "duration parsing failed")
		return
	}
	duration, err := statsKeeper.GetSessionDuration()
	assert.NoError(t, err)
	assert.Equal(t, expectedDuration, duration)
}

func TestGetSessionDurationFailsWhenSessionStartNotMarked(t *testing.T) {
	statsKeeper := NewSessionStatsKeeper(time.Now)

	_, err := statsKeeper.GetSessionDuration()
	assert.EqualError(t, err, "session start was not marked")
}
