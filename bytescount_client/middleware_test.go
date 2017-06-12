package bytescount_client

import (
	"errors"
	"github.com/MysteriumNetwork/node/server"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_Factory(t *testing.T) {
	middleware := NewMiddleware(server.NewClientFake(), "session-test", 1*time.Minute)
	assert.NotNil(t, middleware)
}

func Test_ConsumeLine(t *testing.T) {
	var tests = []struct {
		line             string
		expectedConsumed bool
		expectedError    error
	}{
		{">BYTECOUNT:3018,3264", true, nil},
		{">BYTECOUNT:0,3264", true, nil},
		{">BYTECOUNT:3018,", true, errors.New(`strconv.ParseInt: parsing "": invalid syntax`)},
		{">BYTECOUNT:,", true, errors.New(`strconv.ParseInt: parsing "": invalid syntax`)},
		{"OTHER", false, nil},
		{"BYTECOUNT", false, nil},
		{"BYTECOUNT:", false, nil},
		{"BYTECOUNT:3018,3264", false, nil},
		{">BYTECOUNTT:3018,3264", false, nil},
	}

	middleware := NewMiddleware(server.NewClientFake(), "session-test", 1*time.Minute)
	for _, test := range tests {
		consumed, err := middleware.ConsumeLine(test.line)
		if test.expectedError != nil {
			assert.Error(t, test.expectedError, err.Error(), test.line)
		} else {
			assert.NoError(t, err, test.line)
		}
		assert.Equal(t, test.expectedConsumed, consumed, test.line)
	}
}
