package bytescount

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewSessionStatsSaver(t *testing.T) {
	statsStore := &SessionStatsStore{}

	saver := NewSessionStatsSaver(statsStore)
	stats := SessionStats{BytesSent: 1, BytesReceived: 2}
	saver(stats)
	assert.Equal(t, stats, statsStore.Retrieve())
}
