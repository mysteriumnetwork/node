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
	"strings"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/config"
	nodevent "github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/rs/zerolog/log"
)

var defaultPrice = market.Price{
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
			PerCountry: make(map[string]*market.PriceHistory),
			Defaults: &market.PriceHistory{
				Current: &market.PriceByType{
					Residential: &market.PriceByServiceType{
						Wireguard: &market.Price{
							PricePerHour: defaultPrice.PricePerHour,
							PricePerGiB:  defaultPrice.PricePerGiB,
						},
						Scraping: &market.Price{
							PricePerHour: defaultPrice.PricePerHour,
							PricePerGiB:  defaultPrice.PricePerGiB,
						},
						DataTransfer: &market.Price{
							PricePerHour: defaultPrice.PricePerHour,
							PricePerGiB:  defaultPrice.PricePerGiB,
						},
						DVPN: &market.Price{
							PricePerHour: defaultPrice.PricePerHour,
							PricePerGiB:  defaultPrice.PricePerGiB,
						},
					},
					Other: &market.PriceByServiceType{
						Wireguard: &market.Price{
							PricePerHour: defaultPrice.PricePerHour,
							PricePerGiB:  defaultPrice.PricePerGiB,
						},
						Scraping: &market.Price{
							PricePerHour: defaultPrice.PricePerHour,
							PricePerGiB:  defaultPrice.PricePerGiB,
						},
						DataTransfer: &market.Price{
							PricePerHour: defaultPrice.PricePerHour,
							PricePerGiB:  defaultPrice.PricePerGiB,
						},
						DVPN: &market.Price{
							PricePerHour: defaultPrice.PricePerHour,
							PricePerGiB:  defaultPrice.PricePerGiB,
						},
					},
				},
			},
			CurrentValidUntil: time.Now().Truncate(0).UTC().Add(-time.Hour * 1000),
		},
		discoAPI: discoAPI,
	}
}

// GetCurrentPrice gets the current price from cache if possible, fetches it otherwise.
func (p *Pricer) GetCurrentPrice(nodeType string, country string, serviceType string) (market.Price, error) {
	pricing := p.getPricing()

	price := p.getCurrentByType(pricing, nodeType, country, serviceType)
	if price == nil {
		return market.Price{}, errors.New("no data available")
	}
	return *price, nil
}

func (p *Pricer) getPriceForCountry(pricing market.LatestPrices, country string) *market.PriceHistory {
	v, ok := pricing.PerCountry[strings.ToUpper(country)]
	if ok {
		return v
	}
	return pricing.Defaults
}

func (p *Pricer) getCurrentByType(pricing market.LatestPrices, nodeType string, country string, serviceType string) *market.Price {
	base := p.getPriceForCountry(pricing, country)
	switch strings.ToLower(nodeType) {
	case "residential", "cellular":
		return p.getCurrentByServiceType(base.Current.Residential, serviceType)
	default:
		return p.getCurrentByServiceType(base.Current.Other, serviceType)
	}
}

func (p *Pricer) getCurrentByServiceType(pricingByServiceType *market.PriceByServiceType, serviceType string) *market.Price {
	switch strings.ToLower(serviceType) {
	case "wireguard":
		return pricingByServiceType.Wireguard
	case "scraping":
		return pricingByServiceType.Scraping
	case "dvpn":
		return pricingByServiceType.DVPN
	default:
		return pricingByServiceType.DataTransfer
	}
}

func (p *Pricer) getPreviousByType(pricing market.LatestPrices, nodeType string, country string, serviceType string) *market.Price {
	base := p.getPriceForCountry(pricing, country)
	switch strings.ToLower(nodeType) {
	case "residential", "cellular":
		return p.getCurrentByServiceType(base.Previous.Residential, serviceType)
	default:
		return p.getCurrentByServiceType(base.Previous.Other, serviceType)
	}
}

// IsPriceValid checks if the given price is valid or not.
func (p *Pricer) IsPriceValid(in market.Price, nodeType string, country string, serviceType string) bool {
	if config.GetBool(config.FlagPaymentsDuringSessionDebug) {
		log.Info().Msg("Payments debug bas been enabled, will agree with any price given")
		return true
	}

	pricing := p.getPricing()
	if p.pricesEqual(p.getCurrentByType(pricing, nodeType, country, serviceType), in) {
		return true
	}
	if p.pricesEqual(p.getPreviousByType(pricing, nodeType, country, serviceType), in) {
		return true
	}

	// this is the fallback in case loading of prices fails.
	return p.isCheaperThanDefault(in)
}

func (p *Pricer) pricesEqual(api *market.Price, local market.Price) bool {
	if api == nil || api.PricePerGiB == nil || api.PricePerHour == nil {
		return false
	}

	return api.PricePerGiB.Cmp(local.PricePerGiB) == 0 && api.PricePerHour.Cmp(local.PricePerHour) == 0
}

func (p *Pricer) isCheaperThanDefault(in market.Price) bool {
	return in.PricePerGiB.Cmp(defaultPrice.PricePerGiB) <= 0 && in.PricePerHour.Cmp(defaultPrice.PricePerHour) <= 0
}

// Subscribe subscribes to node events.
func (p *Pricer) Subscribe(bus eventbus.Subscriber) error {
	return bus.SubscribeAsync(nodevent.AppTopicNode, p.preloadOnNodeStart)
}

func (p *Pricer) getPricing() market.LatestPrices {
	p.mut.Lock()
	lastLoad := p.lastLoad
	p.mut.Unlock()

	if time.Now().Truncate(0).UTC().After(lastLoad.CurrentValidUntil) {
		p.loadPricing()
	}

	return p.lastLoad
}

func (p *Pricer) loadPricing() {
	p.mut.Lock()
	defer p.mut.Unlock()

	now := time.Now().Truncate(0).UTC()
	prices, err := p.discoAPI.GetPricing()
	if err != nil {
		log.Err(err).Msg("could not load pricing")
		return
	}
	if prices.Defaults == nil {
		log.Info().Msg("pricing info empty")
		return
	}

	// shift clock skew
	delta := now.Sub(prices.CurrentServerTime)
	prices.CurrentValidUntil = prices.CurrentValidUntil.Add(delta)
	prices.PreviousValidUntil = prices.PreviousValidUntil.Add(delta)
	// equalize
	prices.CurrentServerTime = now

	log.Info().Msgf("pricing info loaded. expires @ %v", prices.CurrentValidUntil)
	p.lastLoad = prices
}

func (p *Pricer) preloadOnNodeStart(se nodevent.Payload) {
	if se.Status != nodevent.StatusStarted {
		return
	}
	p.loadPricing()
}
