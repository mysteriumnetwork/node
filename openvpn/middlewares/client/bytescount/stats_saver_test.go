package bytescount

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewSessionStatsSaver(t *testing.T) {
	statsKeeper := &SessionStatsKeeper{}

	saver := NewSessionStatsSaver(statsKeeper)
	stats := SessionStats{BytesSent: 1, BytesReceived: 2}
	saver(stats)
	assert.Equal(t, stats, statsKeeper.Retrieve())
}
