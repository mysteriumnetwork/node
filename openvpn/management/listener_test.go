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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestListenerShouldAcceptIncomingManagementConnection(t *testing.T) {
	mockedMiddleware := &mockMiddleware{}

	mngmnt := NewManagement(LocalhostOnRandomPort, "[management interface]", mockedMiddleware)
	err := mngmnt.WaitForConnection()
	assert.NoError(t, err)

	_, err = connectTo(mngmnt.BoundAddress)
	assert.NoError(t, err)

	select {
	case connected := <-mngmnt.Connected:
		assert.True(t, connected)
	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "Middleware start method expected to be called in 100 milliseconds")
	}
}

func TestMiddlewareReceivesEventFromManagementInterface(t *testing.T) {
	mockedMiddleware := &mockMiddleware{}
	lineReceived := make(chan string, 1)
	mockedMiddleware.OnLineReceived = func(line string) (bool, error) {
		lineReceived <- line
		return true, nil
	}

	mngmnt := NewManagement(LocalhostOnRandomPort, "[management interface]", mockedMiddleware)
	err := mngmnt.WaitForConnection()
	assert.NoError(t, err)

	mockedOpenvpn, err := connectTo(mngmnt.BoundAddress)
	assert.NoError(t, err)

	err = mockedOpenvpn.Send(">sampleevent\n")
	assert.NoError(t, err)

	select {
	case line := <-lineReceived:
		assert.Equal(t, ">sampleevent", line)
	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "Middleware expected to receive event in 100 milliseconds")
	}
}

func TestMiddlewareCanSendCommandsToManagementInterface(t *testing.T) {
	mockedMiddleware := &mockMiddleware{}
	cmdResult := make(chan string, 1)
	mockedMiddleware.OnStart = func(cmdWriter CommandWriter) error {
		res, _ := cmdWriter.SingleLineCommand("SAMPLECMD")
		cmdResult <- res
		return nil
	}

	mngmnt := NewManagement(LocalhostOnRandomPort, "[management interface]", mockedMiddleware)
	err := mngmnt.WaitForConnection()
	assert.NoError(t, err)

	mockedOpenvpn, err := connectTo(mngmnt.BoundAddress)
	assert.NoError(t, err)

	select {
	case cmd := <-mockedOpenvpn.CmdChan:
		assert.Equal(t, "SAMPLECMD", cmd)
		mockedOpenvpn.Send("SUCCESS: MSG\n")
	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "MockedOpenvpn expected to receive cmd in 100 milliseconds")
	}

	select {
	case res := <-cmdResult:
		assert.Equal(t, "MSG", res)
	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "Middleware expected to receive command result in 100 milliseconds")
	}
}

func TestMiddlewareStartIsCalledOnOpenvpnProcessDisconnect(t *testing.T) {
	mockedMiddleware := &mockMiddleware{}
	startCalled := make(chan bool, 1)
	mockedMiddleware.OnStart = func(writer CommandWriter) error {
		startCalled <- true
		return nil
	}

	mngmnt := NewManagement(LocalhostOnRandomPort, "[management interface]", mockedMiddleware)
	err := mngmnt.WaitForConnection()
	assert.NoError(t, err)

	_, err = connectTo(mngmnt.BoundAddress)
	assert.NoError(t, err)

	select {
	case <-startCalled:
	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "Middleware start method expected to be called in 100 milliseconds")
	}
}

func TestMiddlewareStopIsCalledOnOpenvpnProcessDisconnect(t *testing.T) {
	mockedMiddleware := &mockMiddleware{}
	stopCalled := make(chan bool, 1)
	mockedMiddleware.OnStop = func(cmdWriter CommandWriter) error {
		stopCalled <- true
		return nil
	}

	mngmnt := NewManagement(LocalhostOnRandomPort, "[management interface]", mockedMiddleware)
	err := mngmnt.WaitForConnection()
	assert.NoError(t, err)

	mockedOpenvpn, err := connectTo(mngmnt.BoundAddress)
	assert.NoError(t, err)

	mockedOpenvpn.Send(">STATE: EXITING")
	err = mockedOpenvpn.Disconnect()
	assert.NoError(t, err)

	select {
	case <-stopCalled:
	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "Middleware stop method expected to be called in 100 milliseconds")
	}

}

func TestConnectionChannelReportsFalseWhenListenerIsClosedWithoutConnection(t *testing.T) {
	mngmnt := NewManagement(LocalhostOnRandomPort, "[management interface]", &mockMiddleware{})
	err := mngmnt.WaitForConnection()
	assert.NoError(t, err)

	mngmnt.Stop()

	select {
	case connected := <-mngmnt.Connected:
		assert.False(t, connected)
	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "Expected to receive false on connected channel in 100 milliseconds")
	}
}

func TestListenerShutdownsWhenStopIsCalledAfterConnectionIsEstablished(t *testing.T) {
	mngmnt := NewManagement(LocalhostOnRandomPort, "[management interface]", &mockMiddleware{})
	err := mngmnt.WaitForConnection()
	assert.NoError(t, err)

	_, err = connectTo(mngmnt.BoundAddress)
	assert.NoError(t, err)

	stopFinished := make(chan bool, 1)
	go func() {
		mngmnt.Stop()
		stopFinished <- true
	}()

	select {
	case <-stopFinished:

	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "Management interface expected to stop in 100 milliseconds")
	}

}
