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
	fakeHandler := fakeStatsHandler{}
	compositeHandler := NewCompositeStatsHandler(fakeHandler.save)
	assert.NoError(t, compositeHandler(stats))
	assert.Equal(t, stats, fakeHandler.LastSessionStats)
}

func TestCompositeHandlerWithFailingHandler(t *testing.T) {
	failingHandler := func(stats SessionStats) error { return errors.New("fake error") }
	compositeHandler := NewCompositeStatsHandler(failingHandler)
	assert.Error(t, compositeHandler(stats), "fake error")
}

func TestCompositeHandlerWithMultipleHandlers(t *testing.T) {
	fakeHandler1 := fakeStatsHandler{}
	fakeHandler2 := fakeStatsHandler{}

	compositeHandler := NewCompositeStatsHandler(fakeHandler1.save, fakeHandler2.save)
	assert.NoError(t, compositeHandler(stats))

	assert.Equal(t, stats, fakeHandler1.LastSessionStats)
	assert.Equal(t, stats, fakeHandler2.LastSessionStats)
}
