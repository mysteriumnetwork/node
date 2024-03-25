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
	"github.com/jinzhu/copier"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/client"
)

// NewHermesChannel creates HermesChannel model.
func NewHermesChannel(channelID string, id identity.Identity, hermesID common.Address, channel client.ProviderChannel, promise HermesPromise, beneficiary common.Address) HermesChannel {
	hc := HermesChannel{
		ChannelID:   channelID,
		Identity:    id,
		HermesID:    hermesID,
		Channel:     channel,
		lastPromise: promise,
		Beneficiary: beneficiary,
	}
	return hc
}

// Copy returns a deep copy of channel
func (hc HermesChannel) Copy() HermesChannel {
	var hcCopy HermesChannel
	if err := copier.CopyWithOption(&hcCopy, hc, copier.Option{DeepCopy: true}); err != nil {
		panic(err)
	}
	return hcCopy
}

// HermesChannel represents opened payment channel between identity and hermes.
type HermesChannel struct {
	ChannelID   string
	Identity    identity.Identity
	HermesID    common.Address
	Channel     client.ProviderChannel
	lastPromise HermesPromise
	Beneficiary common.Address
}

// LifetimeBalance returns earnings of all history.
func (hc HermesChannel) LifetimeBalance() *big.Int {
	if hc.lastPromise.Promise.Amount == nil {
		return new(big.Int)
	}
	return hc.lastPromise.Promise.Amount
}

// UnsettledBalance returns current unsettled earnings.
func (hc HermesChannel) UnsettledBalance() *big.Int {
	settled := new(big.Int)
	if hc.Channel.Settled != nil {
		settled = hc.Channel.Settled
	}

	lastPromise := new(big.Int)
	if hc.lastPromise.Promise.Amount != nil {
		lastPromise = hc.lastPromise.Promise.Amount
	}

	return safeSub(lastPromise, settled)
}

func (hc HermesChannel) availableBalance() *big.Int {
	balance := new(big.Int)
	if hc.Channel.Stake != nil {
		balance = hc.Channel.Stake
	}

	settled := new(big.Int)
	if hc.Channel.Settled != nil {
		settled = hc.Channel.Settled
	}

	return new(big.Int).Add(balance, settled)
}

func (hc HermesChannel) balance() *big.Int {
	promised := new(big.Int)
	if hc.lastPromise.Promise.Amount != nil {
		promised = hc.lastPromise.Promise.Amount
	}
	return safeSub(hc.availableBalance(), promised)
}
