/*
 * Copyright (C) 2023 The "MysteriumNetwork/node" Authors.
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

package registry

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/eventbus"
	identity_selector "github.com/mysteriumnetwork/node/identity/selector"
	"github.com/rs/zerolog/log"
)

// FreeRegistrar is responsible for registering default identity for free
type FreeRegistrar struct {
	lock                    sync.Mutex
	selector                identity_selector.Handler
	transactor              transactor
	contractRegistry        IdentityRegistry
	freeRegistrationEnabled bool
}

// NewFreeRegistrar creates new free registrar
func NewFreeRegistrar(selector identity_selector.Handler, transactor transactor, contractRegistry IdentityRegistry, freeRegistrationEnabled bool) *FreeRegistrar {
	return &FreeRegistrar{
		selector:                selector,
		transactor:              transactor,
		contractRegistry:        contractRegistry,
		freeRegistrationEnabled: freeRegistrationEnabled,
	}
}

// Subscribe subscribes to Node events
func (f *FreeRegistrar) Subscribe(eb eventbus.Subscriber) error {
	if !f.freeRegistrationEnabled {
		return nil
	}
	err := eb.SubscribeAsync(event.AppTopicNode, f.handleNodeEvent)
	return err
}

func (f *FreeRegistrar) handleNodeEvent(ev event.Payload) {
	if ev.Status == event.StatusStarted {
		err := f.handleStart()
		if err != nil {
			log.Error().Err(err).Msg("failed to handle free registrar start")
		}
		return
	}
}

func (f *FreeRegistrar) handleStart() error {
	log.Debug().Msg("Try register provider for free")

	chainID := config.GetInt64(config.FlagChainID)
	id, err := f.selector.UseOrCreate("", "", chainID)
	if err != nil {
		log.Error().Err(err).Msg("could not create default identity identity")
		return nil
	}
	status, err := f.contractRegistry.GetRegistrationStatus(chainID, id)
	if err != nil {
		log.Error().Err(err).Msg("could not check registration status")
		return err
	}
	if status == Registered {
		log.Info().Msg("Default identity is already registered")
		return nil
	}

	eligible, err := f.transactor.GetFreeProviderRegistrationEligibility()
	if err != nil {
		return fmt.Errorf("failed to check free registration eligibility: %w", err)
	}

	if !eligible {
		log.Warn().Msg("Free registration is not eligible")
		return nil
	}
	err = f.transactor.RegisterProviderIdentity(id.Address, big.NewInt(0), big.NewInt(0), "", chainID, nil)
	if err != nil {
		return fmt.Errorf("could not register identity: %w", err)
	}
	log.Info().Msgf("Identity created and registered for free: %s", id.Address)
	return nil
}
