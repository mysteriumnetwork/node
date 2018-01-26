package bytescount

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

var stats = SessionStats{BytesSent: 1, BytesReceived: 1}

func TestCompositeHandlerWithNoHandlers(t *testing.T) {
	stats := SessionStats{BytesSent: 1, BytesReceived: 1}

	compositeHandler := NewCompositeStatsHandler()
	assert.NoError(t, compositeHandler(stats))
}

func TestCompositeHandlerWithSuccessfulHandler(t *testing.T) {
	statsRecorder := fakeStatsRecorder{}
	compositeHandler := NewCompositeStatsHandler(statsRecorder.record)
	assert.NoError(t, compositeHandler(stats))
	assert.Equal(t, stats, statsRecorder.LastSessionStats)
}

func TestCompositeHandlerWithFailingHandler(t *testing.T) {
	failingHandler := func(stats SessionStats) error { return errors.New("fake error") }
	compositeHandler := NewCompositeStatsHandler(failingHandler)
	assert.Error(t, compositeHandler(stats), "fake error")
}

func TestCompositeHandlerWithMultipleHandlers(t *testing.T) {
	recorder1 := fakeStatsRecorder{}
	recorder2 := fakeStatsRecorder{}

	compositeHandler := NewCompositeStatsHandler(recorder1.record, recorder2.record)
	assert.NoError(t, compositeHandler(stats))

	assert.Equal(t, stats, recorder1.LastSessionStats)
	assert.Equal(t, stats, recorder2.LastSessionStats)
}
