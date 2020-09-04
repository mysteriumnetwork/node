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
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/payments/client"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
)

type promiseProvider interface {
	Get(channelID string) (HermesPromise, error)
	List(filter HermesPromiseFilter) ([]HermesPromise, error)
}

type channelProvider interface {
	GetProviderChannel(hermesAddress common.Address, addressToCheck common.Address, pending bool) (client.ProviderChannel, error)
}

// HermesChannelRepository is fetches HermesChannel models from blockchain.
type HermesChannelRepository struct {
	promises promiseProvider
	channels channelProvider
}

// NewHermesChannelRepository returns a new instance of HermesChannelRepository.
func NewHermesChannelRepository(promiseProvider promiseProvider, channelProvider channelProvider) *HermesChannelRepository {
	return &HermesChannelRepository{
		promises: promiseProvider,
		channels: channelProvider,
	}
}

// Get retrieves current channel for given identity.
func (hcr *HermesChannelRepository) Get(id identity.Identity, hermesID common.Address) (HermesChannel, error) {
	channelID, err := crypto.GenerateProviderChannelID(id.Address, hermesID.Hex())
	if err != nil {
		return HermesChannel{}, errors.Wrap(err, "could not generate provider channel address")
	}

	channel, err := hcr.fetchChannel(id, hermesID)
	if err != nil {
		return HermesChannel{}, err
	}

	promise, err := hcr.promises.Get(channelID)
	if err != nil && err != ErrNotFound {
		return HermesChannel{}, errors.Wrap(err, fmt.Sprintf("could not get hermes promise for provider %v, hermes %v", id, hermesID.Hex()))
	}

	return NewHermesChannel(id, hermesID, channel, promise.Promise), nil
}

// List retrieves the promise for the given hermes.
func (hcr *HermesChannelRepository) List(filter HermesPromiseFilter) ([]HermesChannel, error) {
	promises, err := hcr.promises.List(filter)
	if err != nil {
		return []HermesChannel{}, err
	}

	result := make([]HermesChannel, len(promises))
	for i, promise := range promises {
		channel, err := hcr.fetchChannel(promise.Identity, promise.HermesID)
		if err != nil {
			return []HermesChannel{}, err
		}

		result[i] = NewHermesChannel(promise.Identity, promise.HermesID, channel, promise.Promise)
	}

	return result, err
}

func (hcr *HermesChannelRepository) fetchChannel(id identity.Identity, hermesID common.Address) (client.ProviderChannel, error) {
	// TODO Should call GetProviderChannelByID() but can't pass pending=false
	channel, err := hcr.channels.GetProviderChannel(hermesID, id.ToCommonAddress(), true)
	if err != nil {
		return client.ProviderChannel{}, errors.Wrap(err, fmt.Sprintf("could not get provider channel for %v, hermes %v", id, hermesID.Hex()))
	}

	return channel, nil
}
