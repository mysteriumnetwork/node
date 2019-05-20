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
	"time"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/core/discovery"
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

	proposalStorage *discovery.ProposalStorage
	proposalChan    chan market.ServiceProposal
}

// NewFetcher create instance of fetcher
func NewFetcher(proposalsStorage *discovery.ProposalStorage, callback FetchCallback, interval time.Duration) *fetcher {
	return &fetcher{
		fetch:         callback,
		fetchInterval: interval,

		proposalStorage: proposalsStorage,
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

func (fetcher *fetcher) SubscribeProposals(proposalsChan chan market.ServiceProposal) {
	fetcher.proposalChan = proposalsChan
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
	fetcher.proposalStorage.AddMultiple(proposals)

	if fetcher.proposalChan != nil {
		for _, proposal := range proposals {
			fetcher.proposalChan <- proposal
		}
	}
	return nil
}
