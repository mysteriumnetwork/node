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
	"github.com/mysteriumnetwork/go-openvpn/openvpn"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/state"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/service_discovery/dto"
	"github.com/mysteriumnetwork/node/session"
)

// DialogCreator creates new dialog between consumer and provider, using given contact information
type DialogCreator func(consumerID, providerID identity.Identity, contact dto.Contact) (communication.Dialog, error)

// VpnClientCreator creates new vpn client by given session,
// consumer identity, provider identity and uses state callback to report state changes
type VpnClientCreator func(session.SessionDto, identity.Identity, identity.Identity, state.Callback, ConnectOptions) (openvpn.Process, error)

// Manager interface provides methods to manage connection
type Manager interface {
	// Connect creates new connection from given consumer to provider, reports error if connection already exists
	Connect(consumerID identity.Identity, providerID identity.Identity, options ConnectOptions) error
	// Status queries current status of connection
	Status() ConnectionStatus
	// Disconnect closes established connection, reports error if no connection
	Disconnect() error
}
