package bytescount

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetSessionStatsStoreReturnsSameInstance(t *testing.T) {
	GetSessionStatsStore().Clear()

	stats := SessionStats{BytesSent: 1, BytesReceived: 2}
	GetSessionStatsStore().Set(stats)
	assert.Equal(t, stats, GetSessionStatsStore().Get())
}
