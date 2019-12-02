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
	"github.com/rs/zerolog/log"
)

// FetchCallback does real fetch of proposals through Mysterium API
type FetchCallback func() ([]market.ServiceProposal, error)

// Fetcher continuously fetches service proposals from discovery service
type Fetcher interface {
	Start() error
	Stop()
}

// Fetcher represents async proposal fetcher from Mysterium API
type fetcher struct {
	fetch         FetchCallback
	fetchInterval time.Duration
	fetchShutdown chan bool

	proposalStorage *discovery.ProposalStorage
}

// NewFetcher create instance of Fetcher
func NewFetcher(proposalsStorage *discovery.ProposalStorage, callback FetchCallback, interval time.Duration) Fetcher {
	return &fetcher{
		fetch:         callback,
		fetchInterval: interval,

		proposalStorage: proposalsStorage,
	}
}

// Start begins fetching proposals to storage
func (fetcher *fetcher) Start() error {
	go func() {
		if err := fetcher.fetchDo(); err != nil {
			log.Warn().Err(err).Msg("Initial proposal fetch failed, continuing")
		}
	}()

	fetcher.fetchShutdown = make(chan bool, 1)
	go fetcher.fetchLoop()

	return nil
}

// Stop ends fetching proposals to storage
func (fetcher *fetcher) Stop() {
	fetcher.fetchShutdown <- true
}

func (fetcher *fetcher) fetchLoop() {
	for {
		select {
		case <-fetcher.fetchShutdown:
			break
		case <-time.After(fetcher.fetchInterval):
			_ = fetcher.fetchDo()
		}
	}
}

func (fetcher *fetcher) fetchDo() error {
	proposals, err := fetcher.fetch()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to fetch proposals")
		return err
	}

	log.Debug().Msgf("Proposals fetched: %d", len(proposals))
	fetcher.proposalStorage.Set(proposals...)
	return nil
}
