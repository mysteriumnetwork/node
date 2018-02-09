package bytescount

import (
	"errors"
	"github.com/mysterium/node/openvpn/middlewares"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_Factory(t *testing.T) {
	statsRecorder := fakeStatsRecorder{}
	middleware := NewMiddleware(statsRecorder.record, 1*time.Minute)
	assert.NotNil(t, middleware)
}

func Test_Start(t *testing.T) {
	statsRecorder := fakeStatsRecorder{}
	middleware := NewMiddleware(statsRecorder.record, 1*time.Minute)
	mockWritter := &middlewares.MockCommandWriter{}
	middleware.Start(mockWritter)
	assert.Equal(t, "bytecount 60", mockWritter.LastLine)
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
		middleware := NewMiddleware(statsRecorder.record, 1*time.Minute)
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
