/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package pingpong

import (
	"errors"
	"sync"
	"time"

	nodevent "github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/rs/zerolog/log"
)

var defaultPrice = market.Prices{
	PricePerHour: crypto.FloatToBigMyst(0.00006),
	PricePerGiB:  crypto.FloatToBigMyst(0.1),
}

type discoAPI interface {
	GetPricing() (market.LatestPrices, error)
}

// Pricer fetches and caches prices from discovery api.
type Pricer struct {
	discoAPI discoAPI
	lastLoad market.LatestPrices
	mut      sync.Mutex
}

// NewPricer creates a new instance of pricer.
func NewPricer(discoAPI discoAPI) *Pricer {
	return &Pricer{
		lastLoad: market.LatestPrices{
			Current: &market.Prices{
				ValidUntil:   time.Now().Add(-time.Hour * 1000),
				PricePerHour: defaultPrice.PricePerHour,
				PricePerGiB:  defaultPrice.PricePerGiB,
			},
		},
		discoAPI: discoAPI,
	}
}

// GetCurrentPrice gets the current price from cache if possible, fetches it otherwise.
func (p *Pricer) GetCurrentPrice() (market.Prices, error) {
	pricing := p.getPricing()
	if pricing.Current != nil {
		return *pricing.Current, nil
	}

	return market.Prices{}, errors.New("could not load pricing info")
}

// IsPriceValid checks if the given price is valid or not.
func (p *Pricer) IsPriceValid(in market.Prices) bool {
	pricing := p.getPricing()
	if p.pricesEqual(pricing.Current, in) {
		return true
	}
	if p.pricesEqual(pricing.Previous, in) {
		return true
	}

	// this is the fallback in case loading of prices fails.
	return p.isCheaperThanDefault(in)
}

func (p *Pricer) pricesEqual(api *market.Prices, local market.Prices) bool {
	if api == nil || api.PricePerGiB == nil || api.PricePerHour == nil {
		return false
	}

	return api.PricePerGiB.Cmp(local.PricePerGiB) == 0 && api.PricePerHour.Cmp(local.PricePerHour) == 0
}

func (p *Pricer) isCheaperThanDefault(in market.Prices) bool {
	return in.PricePerGiB.Cmp(defaultPrice.PricePerGiB) <= 0 && in.PricePerHour.Cmp(defaultPrice.PricePerHour) <= 0
}

// Subscribe subscribes to node events.
func (p *Pricer) Subscribe(bus eventbus.Subscriber) error {
	return bus.SubscribeAsync(nodevent.AppTopicNode, p.preloadOnNodeStart)
}

func (p *Pricer) getPricing() market.LatestPrices {
	if time.Now().UTC().After(p.lastLoad.Current.ValidUntil) {
		p.loadPricing()
	}

	p.mut.Lock()
	defer p.mut.Unlock()

	return p.lastLoad
}

func (p *Pricer) loadPricing() {
	p.mut.Lock()
	defer p.mut.Unlock()
	prices, err := p.discoAPI.GetPricing()
	if err != nil {
		log.Err(err).Msg("could not load pricing")
		return
	}
	log.Info().Msg("pricing info loaded")
	p.lastLoad = prices
}

func (p *Pricer) preloadOnNodeStart(se nodevent.Payload) {
	if se.Status != nodevent.StatusStarted {
		return
	}
	p.loadPricing()
}
