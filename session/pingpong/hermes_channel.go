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

package pingpong

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/client"
	"github.com/mysteriumnetwork/payments/crypto"
)

// HermesChannel represents opened payment channel between identity and hermes.
type HermesChannel struct {
	Identity    identity.Identity
	HermesID    common.Address
	channel     client.ProviderChannel
	lastPromise crypto.Promise
}

// lifetimeBalance returns earnings of all history.
func (hc HermesChannel) lifetimeBalance() *big.Int {
	if hc.lastPromise.Amount == nil {
		return new(big.Int)
	}
	return hc.lastPromise.Amount
}

// unsettledBalance returns current unsettled earnings.
func (hc HermesChannel) unsettledBalance() *big.Int {
	settled := new(big.Int)
	if hc.channel.Settled != nil {
		settled = hc.channel.Settled
	}

	lastPromise := new(big.Int)
	if hc.lastPromise.Amount != nil {
		lastPromise = hc.lastPromise.Amount
	}

	return safeSub(lastPromise, settled)
}

func (hc HermesChannel) availableBalance() *big.Int {
	balance := new(big.Int)
	if hc.channel.Balance != nil {
		balance = hc.channel.Balance
	}

	settled := new(big.Int)
	if hc.channel.Settled != nil {
		settled = hc.channel.Settled
	}

	return new(big.Int).Add(balance, settled)
}

func (hc HermesChannel) balance() *big.Int {
	promised := new(big.Int)
	if hc.lastPromise.Amount != nil {
		promised = hc.lastPromise.Amount
	}
	return safeSub(hc.availableBalance(), promised)
}
