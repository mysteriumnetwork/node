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
	"fmt"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/market"
)

const (
	fetcherLogPrefix = "[proposal-fetcher-api] "
)

// FetchCallback does real fetch of proposals through Mysterium API
type FetchCallback func() ([]market.ServiceProposal, error)

// fetcher represents async proposal fether from Mysterium API
type fetcher struct {
	fetch         FetchCallback
	fetchInterval time.Duration
	fetchShutdown chan bool

	proposalsLock sync.Mutex
	proposals     map[market.ProposalID]market.ServiceProposal
}

// NewFetcher create instance of fetcher
func NewFetcher(callback FetchCallback, interval time.Duration) *fetcher {
	return &fetcher{
		fetch:         callback,
		fetchInterval: interval,

		proposals: make(map[market.ProposalID]market.ServiceProposal, 0),
	}
}

func (fetcher *fetcher) Start() error {
	if err := fetcher.fetchDo(); err != nil {
		return err
	}

	fetcher.fetchShutdown = make(chan bool, 1)
	go fetcher.fetchLoop()

	return nil
}

func (fetcher *fetcher) Stop() {
	fetcher.fetchShutdown <- true
}

func (fetcher *fetcher) GetProposals() map[market.ProposalID]market.ServiceProposal {
	fetcher.proposalsLock.Lock()
	defer fetcher.proposalsLock.Unlock()
	return fetcher.proposals
}

func (fetcher *fetcher) fetchLoop() {
	for {
		select {
		case <-fetcher.fetchShutdown:
			break

		case <-time.After(fetcher.fetchInterval):
			fetcher.fetchDo()
		}
	}
}

func (fetcher *fetcher) fetchDo() error {
	proposals, err := fetcher.fetch()
	if err != nil {
		log.Warn(fetcherLogPrefix, fmt.Sprintf("Failed to fetch proposals: %s", err))
		return err
	}

	log.Info(fetcherLogPrefix, fmt.Sprintf("Proposals fetched: %d", len(proposals)))
	fetcher.proposalsLock.Lock()
	defer fetcher.proposalsLock.Unlock()

	fetcher.proposals = make(map[market.ProposalID]market.ServiceProposal, len(proposals))
	for _, proposal := range proposals {
		fetcher.proposals[proposal.UniqueID()] = proposal
	}
	return nil
}
