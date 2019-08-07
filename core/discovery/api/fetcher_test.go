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

package api

import (
	"sync"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/core/discovery"
	"github.com/mysteriumnetwork/node/market"
	"github.com/stretchr/testify/assert"
)

var (
	proposalFirst    = market.ServiceProposal{ProviderID: "0x1"}
	proposalSecond   = market.ServiceProposal{ProviderID: "0x2"}
	proposalsCurrent = fetchCallback{}
)

type fetchCallback struct {
	proposalsMock []market.ServiceProposal
	mutex         sync.RWMutex
}

func (callback *fetchCallback) Mock(proposals ...market.ServiceProposal) {
	callback.mutex.Lock()
	defer callback.mutex.Unlock()
	callback.proposalsMock = proposals
}

func (callback *fetchCallback) Fetch() ([]market.ServiceProposal, error) {
	callback.mutex.Lock()
	defer callback.mutex.Unlock()
	return callback.proposalsMock, nil
}

func waitForInitialFetch() {
	time.Sleep(2*time.Second + 100*time.Millisecond)
}

func Test_Fetcher_StartFetchesInitialProposals(t *testing.T) {
	storage := discovery.NewStorage()
	fetcher := NewFetcher(storage, proposalsCurrent.Fetch, time.Hour)

	proposalsCurrent.Mock(proposalFirst, proposalSecond)
	err := fetcher.Start()
	defer fetcher.Stop()
	assert.NoError(t, err)

	waitForInitialFetch()

	// Initial fetch should be done after initial delay
	assert.Len(t, storage.Proposals(), 2)
	assert.Exactly(t, []market.ServiceProposal{proposalFirst, proposalSecond}, storage.Proposals())
}

func Test_Fetcher_StartFetchesNewProposals(t *testing.T) {
	storage := discovery.NewStorage()
	fetcher := NewFetcher(storage, proposalsCurrent.Fetch, time.Millisecond)

	err := fetcher.Start()
	defer fetcher.Stop()
	assert.NoError(t, err)

	waitForInitialFetch()

	// Following fetches should be done asynchronously
	proposalsCurrent.Mock(proposalFirst, proposalSecond)
	waitABit()

	assert.Len(t, storage.Proposals(), 2)
	assert.Exactly(t, []market.ServiceProposal{proposalFirst, proposalSecond}, storage.Proposals())
}

func Test_Fetcher_StartNotifiesWithInitialProposals(t *testing.T) {
	storage := discovery.NewStorage()
	fetcher := NewFetcher(storage, proposalsCurrent.Fetch, time.Hour)

	proposalChan := make(chan market.ServiceProposal)
	fetcher.SubscribeProposals(proposalChan)

	proposalsCurrent.Mock(proposalFirst, proposalSecond)
	go func() {
		err := fetcher.Start()
		defer fetcher.Stop()
		assert.NoError(t, err)
	}()

	waitForInitialFetch()

	assert.Exactly(t, proposalFirst, waitForProposal(t, proposalChan))
	assert.Exactly(t, proposalSecond, waitForProposal(t, proposalChan))
}

func Test_Fetcher_StartNotifiesWithNewProposals(t *testing.T) {
	storage := discovery.NewStorage()
	fetcher := NewFetcher(storage, proposalsCurrent.Fetch, time.Millisecond)

	proposalChan := make(chan market.ServiceProposal)
	fetcher.SubscribeProposals(proposalChan)

	proposalsCurrent.Mock(proposalFirst)
	go func() {
		err := fetcher.Start()
		defer fetcher.Stop()
		assert.NoError(t, err)
	}()

	proposalsCurrent.Mock(proposalSecond)
	assert.Exactly(t, proposalSecond, waitForProposal(t, proposalChan))
}

func waitForProposal(t *testing.T, proposalsChan chan market.ServiceProposal) market.ServiceProposal {
	select {
	case proposal := <-proposalsChan:
		return proposal
	case <-time.After(20 * time.Millisecond):
		t.Log("Proposal not fetched")
		return market.ServiceProposal{}
	}
}

func waitABit() {
	//usually time.Sleep call gives a chance for other goroutines to kick in
	//important when testing async code
	time.Sleep(1 * time.Millisecond)
}
