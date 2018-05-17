/*
 * Copyright (C) 2018 The Mysterium Network Authors
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
	"github.com/mysterium/node/openvpn/management"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_Factory(t *testing.T) {
	statsRecorder := fakeStatsRecorder{}
	middleware := NewMiddleware(statsRecorder.record, 1*time.Second)
	assert.NotNil(t, middleware)
}

func Test_Start(t *testing.T) {
	statsRecorder := fakeStatsRecorder{}
	middleware := NewMiddleware(statsRecorder.record, 1*time.Second)
	mockConnection := &management.MockConnection{}
	middleware.Start(mockConnection)
	assert.Equal(t, "bytecount 1", mockConnection.LastLine)
}

func Test_ConsumeLine(t *testing.T) {
	var tests = []struct {
		line                  string
		expectedConsumed      bool
		expectedError         error
		expectedBytesReceived int
		expectedBytesSent     int
	}{
		{">BYTECOUNT:3018,3264", true, nil, 3018, 3264},
		{">BYTECOUNT:0,3264", true, nil, 0, 3264},
		{">BYTECOUNT:3018,", true, errors.New(`strconv.ParseInt: parsing "": invalid syntax`), 0, 0},
		{">BYTECOUNT:,", true, errors.New(`strconv.ParseInt: parsing "": invalid syntax`), 0, 0},
		{"OTHER", false, nil, 0, 0},
		{"BYTECOUNT", false, nil, 0, 0},
		{"BYTECOUNT:", false, nil, 0, 0},
		{"BYTECOUNT:3018,3264", false, nil, 0, 0},
		{">BYTECOUNTT:3018,3264", false, nil, 0, 0},
	}

	for _, test := range tests {
		statsRecorder := &fakeStatsRecorder{}
		middleware := NewMiddleware(statsRecorder.record, 1*time.Second)
		consumed, err := middleware.ConsumeLine(test.line)
		if test.expectedError != nil {
			assert.Error(t, test.expectedError, err.Error(), test.line)
		} else {
			assert.NoError(t, err, test.line)
		}
		assert.Equal(t, test.expectedConsumed, consumed, test.line)
		assert.Equal(t, test.expectedBytesReceived, statsRecorder.LastSessionStats.BytesReceived)
		assert.Equal(t, test.expectedBytesSent, statsRecorder.LastSessionStats.BytesSent)
	}
}
