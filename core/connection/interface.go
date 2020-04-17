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
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
)

// DialogCreator creates new dialog between consumer and provider, using given contact information
type DialogCreator func(consumerID, providerID identity.Identity, contact market.Contact) (communication.Dialog, error)

// ConsumerConfig are the parameters used for the initiation of connection
type ConsumerConfig interface{}

// Connection represents a connection
type Connection interface {
	Start(context.Context, ConnectOptions) error
	Wait() error
	Stop()
	GetConfig() (ConsumerConfig, error)
	State() <-chan State
	Statistics() (Statistics, error)
}

// StateChannel is the channel we receive state change events on
type StateChannel chan State

// PromiseIssuer issues promises from consumer to provider.
// Consumer signs those promises.
type PromiseIssuer interface {
	Start(proposal market.ServiceProposal) error
	Stop() error
}

// PromiseIssuerCreator creates new PromiseIssuer given context
type PromiseIssuerCreator func(issuerID identity.Identity, dialog communication.Dialog) PromiseIssuer

// Manager interface provides methods to manage connection
type Manager interface {
	// Connect creates new connection from given consumer to provider, reports error if connection already exists
	Connect(consumerID identity.Identity, accountantID common.Address, proposal market.ServiceProposal, params ConnectParams) error
	// Status queries current status of connection
	Status() Status
	// Disconnect closes established connection, reports error if no connection
	Disconnect() error
}
