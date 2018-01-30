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
	assert.NoError(t, err)
	duration := statsKeeper.GetSessionDuration()
	assert.Equal(t, expectedDuration, duration)
}

func TestGetSessionDurationFailsWhenSessionStartNotMarked(t *testing.T) {
	statsKeeper := NewSessionStatsKeeper(time.Now)

	assert.Equal(t, time.Duration(0), statsKeeper.GetSessionDuration())
}
