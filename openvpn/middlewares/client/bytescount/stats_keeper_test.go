package bytescount

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSessionStatsStoreWorks(t *testing.T) {
	statsKeeper := &SessionStatsKeeper{}
	stats := SessionStats{BytesSent: 1, BytesReceived: 2}

	statsKeeper.Save(stats)
	assert.Equal(t, stats, statsKeeper.Retrieve())
}
