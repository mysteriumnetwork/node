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

// finderAPI represents async proposal fetcher from Mysterium API
type finderAPI struct {
	fetch         FetchCallback
	fetchInterval time.Duration
	fetchShutdown chan bool

	proposalStorage *discovery.ProposalStorage
}

// NewFinder create instance of API finder
func NewFinder(proposalsStorage *discovery.ProposalStorage, callback FetchCallback, interval time.Duration) *finderAPI {
	return &finderAPI{
		fetch:         callback,
		fetchInterval: interval,

		proposalStorage: proposalsStorage,
	}
}

// Start begins fetching proposals to storage
func (fa *finderAPI) Start() error {
	go func() {
		if err := fa.fetchDo(); err != nil {
			log.Warn().Err(err).Msg("Initial proposal fetch failed, continuing")
		}
	}()

	fa.fetchShutdown = make(chan bool, 1)
	go fa.fetchLoop()

	return nil
}

// Stop ends fetching proposals to storage
func (fa *finderAPI) Stop() {
	fa.fetchShutdown <- true
}

func (fa *finderAPI) fetchLoop() {
	for {
		select {
		case <-fa.fetchShutdown:
			break
		case <-time.After(fa.fetchInterval):
			_ = fa.fetchDo()
		}
	}
}

func (fa *finderAPI) fetchDo() error {
	proposals, err := fa.fetch()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to fetch proposals")
		return err
	}

	log.Debug().Msgf("Proposals fetched: %d", len(proposals))
	fa.proposalStorage.AddProposal(proposals...)
	return nil
}
