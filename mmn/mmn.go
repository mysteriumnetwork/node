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

package mmn

import (
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
)

// MMN struct
type MMN struct {
	collector *Collector
	client    *client
}

// NewMMN creates new instance of MMN
func NewMMN(collector *Collector, client *client) *MMN {
	return &MMN{collector, client}
}

// Subscribe subscribes to node events and reports them to MMN
func (m *MMN) Subscribe(eventBus eventbus.EventBus) error {
	err := eventBus.SubscribeAsync(
		identity.AppTopicIdentityUnlock,
		m.handleRegistration,
	)
	if err != nil {
		return err
	}

	return nil
}

func (m *MMN) handleRegistration(identity string) {
	if err := m.register(identity); err != nil {
		log.Error().Msgf("Failed to register to MMN: %v", err)
	}
}

func (m *MMN) register(identity string) error {
	m.collector.SetIdentity(identity)

	return m.client.RegisterNode(m.collector.GetCollectedInformation())
}
