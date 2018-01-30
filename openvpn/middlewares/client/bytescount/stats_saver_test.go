package bytescount

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewSessionStatsSaver(t *testing.T) {
	statsKeeper := NewSessionStatsKeeper(time.Now)

	saver := NewSessionStatsSaver(statsKeeper)
	stats := SessionStats{BytesSent: 1, BytesReceived: 2}
	saver(stats)
	assert.Equal(t, stats, statsKeeper.Retrieve())
}
