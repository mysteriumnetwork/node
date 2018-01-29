package bytescount

import (
	"github.com/mysterium/node/utils"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestStatsSavingWorks(t *testing.T) {
	statsKeeper := NewSessionStatsKeeper(time.Now)
	stats := SessionStats{BytesSent: 1, BytesReceived: 2}

	statsKeeper.Save(stats)
	assert.Equal(t, stats, statsKeeper.Retrieve())
}

func TestGetSessionDurationReturnsFlooredDuration(t *testing.T) {
	settableClock := utils.SettableClock{}
	statsKeeper := NewSessionStatsKeeper(settableClock.GetTime)

	settableClock.SetTime(time.Date(2000, time.January, 0, 10, 12, 3, 0, time.UTC))
	statsKeeper.MarkSessionStart()

	settableClock.SetTime(time.Date(2000, time.January, 0, 10, 12, 4, 700000000, time.UTC))
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
