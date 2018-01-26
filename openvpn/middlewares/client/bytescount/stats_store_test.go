package bytescount

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSessionStatsStoreWorks(t *testing.T) {
	statsStore := &SessionStatsStore{}
	stats := SessionStats{BytesSent: 1, BytesReceived: 2}

	statsStore.Save(stats)
	assert.Equal(t, stats, statsStore.Retrieve())
}
