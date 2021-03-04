/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package pilvytis

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
)

type identityProvider interface {
	GetIdentities() []identity.Identity
}

type statusTracker interface {
	Track()
	Pause()
}

// Service payment service Pilvytis.
type Service struct {
	api              *API
	eventBus         eventbus.Publisher
	identityProvider identityProvider
	tracker          statusTracker
}

// NewService creates a Service instance.
func NewService(api *API, identityProvider identityProvider, tracker statusTracker) *Service {
	return &Service{api: api, identityProvider: identityProvider, tracker: tracker}
}

// Start starts Service workers.
func (s *Service) Start() {
	s.tracker.Track()
}

func (s *Service) validateIdentity(id identity.Identity) error {
	for _, idd := range s.identityProvider.GetIdentities() {
		if idd.Address == id.Address {
			return nil
		}
	}
	return fmt.Errorf("could not find the identity %s", id.Address)
}

// CreateOrder creates a payment order.
func (s *Service) CreateOrder(id identity.Identity, mystAmount float64, payCurrency string, lightning bool) (*OrderResponse, error) {
	if err := s.validateIdentity(id); err != nil {
		return nil, err
	}
	order, err := s.api.CreatePaymentOrder(id, mystAmount, payCurrency, lightning)
	if err != nil {
		return nil, err
	}
	s.tracker.Track()
	return order, nil
}

// GetOrder gets order by ID.
func (s *Service) GetOrder(id identity.Identity, orderID uint64) (*OrderResponse, error) {
	if err := s.validateIdentity(id); err != nil {
		return nil, err
	}
	return s.api.GetPaymentOrder(id, orderID)
}

// ListOrders lists payment orders.
func (s *Service) ListOrders(id identity.Identity) ([]OrderResponse, error) {
	if err := s.validateIdentity(id); err != nil {
		return nil, err
	}
	return s.api.GetPaymentOrders(id)
}

// Currencies returns supported currencies.
func (s *Service) Currencies() ([]string, error) {
	return s.api.GetPaymentOrderCurrencies()
}

// ExchangeRate returns rate of MYST in quote currency.
func (s *Service) ExchangeRate(quote string) (float64, error) {
	rates, err := s.api.GetMystExchangeRate()
	if err != nil {
		return 0, err
	}
	rate, ok := rates[strings.ToUpper(quote)]
	if !ok {
		return 0, errors.New("currency not supported")
	}
	return rate, nil
}

// Stop stops the Service workers.
func (s *Service) Stop() {
	s.tracker.Pause()
}
