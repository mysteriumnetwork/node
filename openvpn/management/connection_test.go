package management

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSingleOutputCommandHandlesSuccess(t *testing.T) {
	mockWriter := &mockWriter{}
	outputChannel := make(chan string, 1)
	conn := newSocketConnection(mockWriter, outputChannel)
	outputChannel <- "SUCCESS: message"

	success, err := conn.SingleOutputCommand("template: %d", 123)
	assert.NoError(t, err)
	assert.Equal(t, "message", success)
	assert.Equal(t, "template: 123\n", mockWriter.receivedCommand)
}

func TestSingleOutputCommandHandlesFailure(t *testing.T) {
	mockWriter := &mockWriter{}
	outputChannel := make(chan string, 1)
	conn := newSocketConnection(mockWriter, outputChannel)
	outputChannel <- "ERROR: error"

	success, err := conn.SingleOutputCommand("anything")
	assert.Empty(t, success)
	assert.Equal(t, errors.New("command error: error"), err)
}

func TestSingleOutputCommandHandlesUnknownResponse(t *testing.T) {
	mockWriter := &mockWriter{}
	outputChannel := make(chan string, 1)
	conn := newSocketConnection(mockWriter, outputChannel)
	outputChannel <- "200 OK HTTP/1.1"

	success, err := conn.SingleOutputCommand("anything")
	assert.Empty(t, success)
	assert.Equal(t, errors.New("unknown command response: 200 OK HTTP/1.1"), err)

}

func TestMultipleOutputCommandHandlesResults(t *testing.T) {

	mockWriter := &mockWriter{}
	outputChannel := make(chan string, 1)
	conn := newSocketConnection(mockWriter, outputChannel)
	go func() {
		outputChannel <- "SUCCESS: great"
		outputChannel <- "This is"
		outputChannel <- "Multiline cmd output"
		outputChannel <- "END"
	}()

	success, output, err := conn.MultiOutputCommand("test: %s , %d", "value", 123)
	assert.NoError(t, err)
	assert.Equal(t, "test: value , 123\n", mockWriter.receivedCommand)
	assert.Equal(t, "great", success)
	assert.Equal(
		t,
		[]string{
			"This is",
			"Multiline cmd output",
		},
		output,
	)

}

type mockWriter struct {
	receivedCommand string
}

func (mw *mockWriter) Write(buff []byte) (int, error) {
	mw.receivedCommand = string(buff)
	return len(buff), nil
}
