package bytescount

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

type fakeConnection struct {
	lastDataWritten []byte
}

func (conn *fakeConnection) Read(b []byte) (int, error) {
	return 0, nil
}

func (conn *fakeConnection) Write(b []byte) (n int, err error) {
	conn.lastDataWritten = b
	return 0, nil
}

func (conn *fakeConnection) Close() error {
	return nil
}

func (conn *fakeConnection) LocalAddr() net.Addr {
	return nil
}

func (conn *fakeConnection) RemoteAddr() net.Addr {
	return nil
}

func (conn *fakeConnection) SetDeadline(t time.Time) error {
	return nil
}

func (conn *fakeConnection) SetReadDeadline(t time.Time) error {
	return nil
}

func (conn *fakeConnection) SetWriteDeadline(t time.Time) error {
	return nil
}

type fakeStatsSender struct {
	lastBytesSent, lastBytesReceived int
}

func (sender *fakeStatsSender) send(bytesSent, bytesReceived int) error {
	sender.lastBytesSent = bytesSent
	sender.lastBytesReceived = bytesReceived
	return nil
}

func Test_Factory(t *testing.T) {
	statsSender := fakeStatsSender{}
	middleware := NewMiddleware(statsSender.send, 1*time.Minute)
	assert.NotNil(t, middleware)
}

func Test_Start(t *testing.T) {
	statsSender := fakeStatsSender{}
	middleware := NewMiddleware(statsSender.send, 1*time.Minute)
	connection := &fakeConnection{}
	middleware.Start(connection)
	assert.Equal(t, []byte("bytecount 60\n"), connection.lastDataWritten)
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
		statsSender := &fakeStatsSender{}
		middleware := NewMiddleware(statsSender.send, 1*time.Minute)
		consumed, err := middleware.ConsumeLine(test.line)
		if test.expectedError != nil {
			assert.Error(t, test.expectedError, err.Error(), test.line)
		} else {
			assert.NoError(t, err, test.line)
		}
		assert.Equal(t, test.expectedConsumed, consumed, test.line)
		assert.Equal(t, test.expectedBytesReceived, statsSender.lastBytesReceived)
		assert.Equal(t, test.expectedBytesSent, statsSender.lastBytesSent)
	}
}
