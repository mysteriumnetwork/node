/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package service

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/go-openvpn/openvpn"
)

func Test_RestartingServerStartsAndStops(t *testing.T) {
	server := &restartingServer{
		stop:                make(chan struct{}),
		waiter:              make(chan error),
		natPinger:           &MockNATPinger{},
		openvpnFactory:      MockOpenvpnFactory,
		lastSessionShutdown: make(chan bool),
	}
	err := server.Start()
	assert.Nil(t, err)
	go func() {
		server.Stop()
	}()
	err = server.Wait()
	assert.Nil(t, err)
}

func Test_RestartingServerExitsOnOpenvpnStartFail(t *testing.T) {
	openvpn := &MockOpenvpnProcess{
		stop:       make(chan struct{}),
		startError: errors.New("some error"),
	}
	factory := &InjectableMockOpenvpnFactory{
		proc: openvpn,
	}
	server := &restartingServer{
		stop:                make(chan struct{}),
		waiter:              make(chan error),
		natPinger:           &MockNATPinger{},
		openvpnFactory:      factory.MockFactory,
		lastSessionShutdown: make(chan bool),
	}

	err := server.Start()
	assert.Nil(t, err)
	err = server.Wait()
	assert.Equal(t, openvpn.startError, err)
	assert.Equal(t, 1, factory.factoryCalls)
}

func Test_RestartingServerExitsOnOpenvpnWaitFail(t *testing.T) {
	openvpn := &MockOpenvpnProcess{
		stop:      make(chan struct{}),
		waitError: errors.New("some error"),
	}
	factory := &InjectableMockOpenvpnFactory{
		proc: openvpn,
	}
	server := &restartingServer{
		stop:                make(chan struct{}),
		waiter:              make(chan error),
		natPinger:           &MockNATPinger{},
		openvpnFactory:      factory.MockFactory,
		lastSessionShutdown: make(chan bool),
	}

	err := server.Start()
	assert.Nil(t, err)

	openvpn.Stop()

	err = server.Wait()
	assert.Equal(t, openvpn.waitError, err)
	assert.Equal(t, 1, factory.factoryCalls)
}

func Test_ServerRestartsIfLastSession(t *testing.T) {
	openvpn := &MockOpenvpnProcess{
		stop: make(chan struct{}),
	}
	factory := &InjectableMockOpenvpnFactory{
		proc: openvpn,
	}
	server := &restartingServer{
		stop:                make(chan struct{}),
		waiter:              make(chan error),
		natPinger:           &MockNATPinger{},
		openvpnFactory:      factory.MockFactory,
		lastSessionShutdown: make(chan bool),
	}

	err := server.Start()
	assert.Nil(t, err)

	go func() {
		server.lastSessionShutdown <- true
		time.Sleep(time.Millisecond * 20)
		server.lastSessionShutdown <- true
		time.Sleep(time.Millisecond * 20)
		server.Stop()
	}()

	err = server.Wait()
	assert.Nil(t, err)

	assert.Equal(t, 3, factory.factoryCalls)
}

type InjectableMockOpenvpnFactory struct {
	proc         openvpn.Process
	factoryCalls int
}

func (imop *InjectableMockOpenvpnFactory) MockFactory() openvpn.Process {
	imop.factoryCalls++
	return imop.proc
}

func MockOpenvpnFactory() openvpn.Process {
	return &MockOpenvpnProcess{
		stop: make(chan struct{}),
	}
}

type MockOpenvpnProcess struct {
	stop       chan struct{}
	startError error
	waitError  error
}

func (mop *MockOpenvpnProcess) Start() error {
	return mop.startError
}

func (mop *MockOpenvpnProcess) Wait() error {
	<-mop.stop
	return mop.waitError
}

func (mop *MockOpenvpnProcess) Stop() {
	mop.stop <- struct{}{}
}

type MockNATPinger struct{}

func (mnp *MockNATPinger) BindProvider(port int) {

}

func (mnp *MockNATPinger) WaitForHole() error {
	return nil
}
