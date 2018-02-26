package bytescount

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewSelectiveStatsHandlerEach(t *testing.T) {
	statsRecorder := fakeStatsRecorder{}
	handler, _ := NewSelectiveStatsHandler(statsRecorder.record, 1)
	stats := SessionStats{
		BytesSent:     1,
		BytesReceived: 2,
	}

	handler(stats)
	assert.Equal(t, stats, statsRecorder.LastSessionStats)
}

func TestNewSelectiveStatsHandlerEveryThird(t *testing.T) {
	statsRecorder := fakeStatsRecorder{}
	handler, _ := NewSelectiveStatsHandler(statsRecorder.record, 3)

	stats := SessionStats{
		BytesSent:     1,
		BytesReceived: 2,
	}
	emptyStats := SessionStats{}

	handler(stats)
	assert.Equal(t, stats, statsRecorder.LastSessionStats)

	statsRecorder.LastSessionStats = emptyStats

	handler(stats)
	assert.Equal(t, emptyStats, statsRecorder.LastSessionStats)
	handler(stats)
	handler(stats)
	assert.Equal(t, stats, statsRecorder.LastSessionStats)

	statsRecorder.LastSessionStats = emptyStats

	handler(stats)
	assert.Equal(t, emptyStats, statsRecorder.LastSessionStats)
	handler(stats)
	handler(stats)
	assert.Equal(t, stats, statsRecorder.LastSessionStats)
}

func TestNewSelectiveStatsHandlerInvalidValues(t *testing.T) {
	statsRecorder := fakeStatsRecorder{}

	_, err := NewSelectiveStatsHandler(statsRecorder.record, 0)
	assert.EqualError(t, err, "Invalid 'times' parameter")

	_, err = NewSelectiveStatsHandler(statsRecorder.record, -1)
	assert.EqualError(t, err, "Invalid 'times' parameter")
}
