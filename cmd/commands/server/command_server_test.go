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

package server

import (
	"github.com/mysterium/node/server"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

var (
	activeProposal = dto_discovery.ServiceProposal{}
)

func TestProposalUnregisteredWhenPingerClosed(t *testing.T) {
	stopPinger := make(chan int)
	fakeDiscoveryClient := server.NewClientFake()
	fakeDiscoveryClient.RegisterProposal(activeProposal, nil)

	finished := make(chan bool)
	fakeCmd := Command{WaitUnregister: &sync.WaitGroup{}}
	fakeCmd.WaitUnregister.Add(1)

	go func() {
		fakeCmd.pingProposalLoop(activeProposal, fakeDiscoveryClient, nil, stopPinger)
		finished <- true
	}()

	close(stopPinger) //causes proposal to be unregistered

	select {
	case _ = <-finished:
		proposals, err := fakeDiscoveryClient.FindProposals(activeProposal.ProviderID)

		assert.NoError(t, err)
		assert.Len(t, proposals, 0)
	case <-time.After(500 * time.Millisecond):
		assert.Fail(t, "failed to stop pinger")
	}
}
