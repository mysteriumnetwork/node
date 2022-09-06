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

package event

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/mysteriumnetwork/node/consumer/bandwidth"
	"github.com/mysteriumnetwork/node/consumer/session"
	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/datasize"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/session/pingpong"
	pingpongEvent "github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/payments/crypto"
)

// AppTopicState is the topic that we use to announce state changes to via the event bus
const AppTopicState = "State change"

// State represents the node state at the current moment. It's a read only object, used only to display data.
type State struct {
	Services         []contract.ServiceInfoDTO
	Sessions         []session.History
	Connections      map[string]Connection
	Identities       []Identity
	ProviderChannels []pingpong.HermesChannel
}

// Identity represents identity and its status.
type Identity struct {
	Address            string
	RegistrationStatus registry.RegistrationStatus
	ChannelAddress     common.Address

	Balance           *big.Int
	Earnings          *big.Int
	EarningsTotal     *big.Int
	EarningsPerHermes map[common.Address]pingpongEvent.Earnings

	HermesID common.Address
}

// Connection represents consumer connection state.
type Connection struct {
	Session    connectionstate.Status
	Statistics connectionstate.Statistics
	Throughput bandwidth.Throughput
	Invoice    crypto.Invoice
}

func (c Connection) String() string {
	spent := money.New(big.NewInt(0))
	if c.Invoice.AgreementTotal != nil {
		spent = money.New(c.Invoice.AgreementTotal)
	}

	return fmt.Sprintf(
		"ID: %s, state: %s, duration: %s data: %s/%s, throughput: %s/%s, spent: %s",
		c.Session.SessionID,
		c.Session.State,
		c.Session.Duration(),
		datasize.FromBytes(c.Statistics.BytesReceived),
		datasize.FromBytes(c.Statistics.BytesSent),
		c.Throughput.Down,
		c.Throughput.Up,
		spent,
	)
}
