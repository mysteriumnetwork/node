/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

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
