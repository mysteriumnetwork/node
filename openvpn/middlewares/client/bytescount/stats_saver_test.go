package bytescount

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewSessionStatsSaver(t *testing.T) {
	GetSessionStatsStore().Clear()

	saver := NewSessionStatsSaver()
	stats := SessionStats{BytesSent: 1, BytesReceived: 2}
	saver(stats)
	assert.Equal(t, stats, GetSessionStatsStore().Get())
}
