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

package management

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSingleOutputCommandHandlesSuccess(t *testing.T) {
	mockWriter := &mockWriter{}
	outputChannel := make(chan string, 1)
	conn := newChannelConnection(mockWriter, outputChannel)
	outputChannel <- "SUCCESS: message"

	success, err := conn.SingleLineCommand("template: %d", 123)
	assert.NoError(t, err)
	assert.Equal(t, "message", success)
	assert.Equal(t, "template: 123\n", mockWriter.receivedCommand)
}

func TestSingleOutputCommandHandlesFailure(t *testing.T) {
	mockWriter := &mockWriter{}
	outputChannel := make(chan string, 1)
	conn := newChannelConnection(mockWriter, outputChannel)
	outputChannel <- "ERROR: error"

	success, err := conn.SingleLineCommand("anything")
	assert.Empty(t, success)
	assert.Equal(t, errors.New("command error: error"), err)
}

func TestSingleOutputCommandHandlesUnknownResponse(t *testing.T) {
	mockWriter := &mockWriter{}
	outputChannel := make(chan string, 1)
	conn := newChannelConnection(mockWriter, outputChannel)
	outputChannel <- "200 OK HTTP/1.1"

	success, err := conn.SingleLineCommand("anything")
	assert.Empty(t, success)
	assert.Equal(t, errors.New("unknown command response: 200 OK HTTP/1.1"), err)

}

func TestMultipleOutputCommandHandlesResults(t *testing.T) {

	mockWriter := &mockWriter{}
	outputChannel := make(chan string, 1)
	conn := newChannelConnection(mockWriter, outputChannel)
	go func() {
		outputChannel <- "SUCCESS: great"
		outputChannel <- "This is"
		outputChannel <- "Multiline cmd output"
		outputChannel <- "END"
	}()

	success, output, err := conn.MultiLineCommand("test: %s , %d", "value", 123)
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
