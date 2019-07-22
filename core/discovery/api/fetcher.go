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
	"time"

	"github.com/mysteriumnetwork/node/core/discovery"
	"github.com/mysteriumnetwork/node/market"
	"github.com/pkg/errors"
)

// FetchCallback does real fetch of proposals through Mysterium API
type FetchCallback func() ([]market.ServiceProposal, error)

// Fetcher represents async proposal fetcher from Mysterium API
type Fetcher struct {
	fetch         FetchCallback
	fetchInterval time.Duration
	fetchShutdown chan bool

	proposalStorage       *discovery.ProposalStorage
	proposalSubscriptions []chan market.ServiceProposal
}

// NewFetcher create instance of Fetcher
func NewFetcher(proposalsStorage *discovery.ProposalStorage, callback FetchCallback, interval time.Duration) *Fetcher {
	return &Fetcher{
		fetch:         callback,
		fetchInterval: interval,

		proposalStorage:       proposalsStorage,
		proposalSubscriptions: make([]chan market.ServiceProposal, 0),
	}
}

// Start begins fetching proposals to storage
func (fetcher *Fetcher) Start() error {
	go func() {
		// FIXME: fix via mobile DI (remove override* methods and configure tunnels via node.Boostrap()?)
		// Add 2 sec delay to complete service startup due to mobile DI flow being a bit different:
		// service definitions are registered via `OverrideOpenvpnConnection`.
		// Definitions must be available at the time of the fetch, otherwise valid proposals will be discarded
		// and user will have to wait for another 30 seconds for them to be populated.
		time.Sleep(2 * time.Second)
		if err := fetcher.fetchDo(); err != nil {
			log.Warn(errors.Wrap(err, "initial proposal fetch failed, continuing"))
		}
	}()

	fetcher.fetchShutdown = make(chan bool, 1)
	go fetcher.fetchLoop()

	return nil
}

// Stop ends fetching proposals to storage
func (fetcher *Fetcher) Stop() {
	fetcher.fetchShutdown <- true
}

// SubscribeProposals allows to subscribe all fetched proposals
func (fetcher *Fetcher) SubscribeProposals(proposalsChan chan market.ServiceProposal) {
	fetcher.proposalSubscriptions = append(fetcher.proposalSubscriptions, proposalsChan)
}

func (fetcher *Fetcher) fetchLoop() {
	for {
		select {
		case <-fetcher.fetchShutdown:
			break
		case <-time.After(fetcher.fetchInterval):
			fetcher.fetchDo()
		}
	}
}

func (fetcher *Fetcher) fetchDo() error {
	proposals, err := fetcher.fetch()
	if err != nil {
		log.Warnf("failed to fetch proposals: %s", err)
		return err
	}

	log.Infof("proposals fetched: %d", len(proposals))
	fetcher.proposalStorage.Set(proposals...)

	for _, proposal := range proposals {
		for _, subscription := range fetcher.proposalSubscriptions {
			subscription <- proposal
		}
	}
	return nil
}
