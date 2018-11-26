/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package connection

import (
	stats_dto "github.com/mysteriumnetwork/node/client/stats/dto"
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/service_discovery/dto"
)

// DialogCreator creates new dialog between consumer and provider, using given contact information
type DialogCreator func(consumerID, providerID identity.Identity, contact dto.Contact) (communication.Dialog, error)

// Connection represents a connection
type Connection interface {
	Start() error
	Wait() error
	Stop()
}

// StateChannel is the channel we receive state change events on
type StateChannel chan State

// StatisticsChannel is the channel we receive stats change events on
type StatisticsChannel chan stats_dto.SessionStats

// PromiseIssuer issues promises from consumer to provider.
// Consumer signs those promises.
type PromiseIssuer interface {
	Start(proposal dto.ServiceProposal) error
	Stop() error
}

// PromiseIssuerCreator creates new PromiseIssuer given context
type PromiseIssuerCreator func(issuerID identity.Identity, dialog communication.Dialog) PromiseIssuer

// Manager interface provides methods to manage connection
type Manager interface {
	// Connect creates new connection from given consumer to provider, reports error if connection already exists
	Connect(consumerID identity.Identity, proposal dto.ServiceProposal, params ConnectParams) error
	// Status queries current status of connection
	Status() ConnectionStatus
	// Disconnect closes established connection, reports error if no connection
	Disconnect() error
}
