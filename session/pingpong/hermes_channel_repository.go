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
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/config"
	nodevent "github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
	pinge "github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/payments/client"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/rs/zerolog/log"
)

type promiseProvider interface {
	Get(chainID int64, channelID string) (HermesPromise, error)
	List(filter HermesPromiseFilter) ([]HermesPromise, error)
}

type channelProvider interface {
	GetProviderChannel(chainID int64, hermesAddress common.Address, addressToCheck common.Address, pending bool) (client.ProviderChannel, error)
}

type beneficiaryProvider interface {
	GetBeneficiary(identity common.Address) (common.Address, error)
}

// HermesChannelRepository is fetches HermesChannel models from blockchain.
type HermesChannelRepository struct {
	promiseProvider promiseProvider
	channelProvider channelProvider
	publisher       eventbus.Publisher
	channels        map[int64][]HermesChannel
	bprovider       beneficiaryProvider
	lock            sync.RWMutex
}

// NewHermesChannelRepository returns a new instance of HermesChannelRepository.
func NewHermesChannelRepository(promiseProvider promiseProvider, channelProvider channelProvider, publisher eventbus.Publisher, bprovider beneficiaryProvider) *HermesChannelRepository {
	return &HermesChannelRepository{
		promiseProvider: promiseProvider,
		channelProvider: channelProvider,
		publisher:       publisher,
		bprovider:       bprovider,
		channels:        make(map[int64][]HermesChannel, 0),
	}
}

// Fetch force identity's channel update and returns updated channel.
func (hcr *HermesChannelRepository) Fetch(chainID int64, id identity.Identity, hermesID common.Address) (HermesChannel, error) {
	hcr.lock.Lock()
	defer hcr.lock.Unlock()

	channelID, err := crypto.GenerateProviderChannelID(id.Address, hermesID.Hex())
	if err != nil {
		return HermesChannel{}, fmt.Errorf("could not generate provider channel address: %w", err)
	}

	promise, err := hcr.promiseProvider.Get(chainID, channelID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return HermesChannel{}, fmt.Errorf("could not get hermes promise for provider %v, hermes %v: %w", id, hermesID.Hex(), err)
	}

	channel, err := hcr.fetchChannel(chainID, promise.ChannelID, id, hermesID, promise)
	if err != nil {
		return HermesChannel{}, err
	}

	return channel, nil
}

// ErrUnknownChain is returned when an operation cannot be completed because
// the given chain is unknown or isn't configured.
var ErrUnknownChain = errors.New("unknown chain")

// Get retrieves identity's channel with given hermes.
func (hcr *HermesChannelRepository) Get(chainID int64, id identity.Identity, hermesID common.Address) (HermesChannel, bool) {
	hcr.lock.RLock()
	defer hcr.lock.RUnlock()

	v, ok := hcr.channels[chainID]
	if !ok {
		return HermesChannel{}, false
	}

	for _, channel := range v {
		if channel.Identity == id && channel.HermesID == hermesID {
			return channel, true
		}
	}

	return HermesChannel{}, false
}

// List retrieves identity's channels with all known hermeses.
func (hcr *HermesChannelRepository) List(chainID int64) []HermesChannel {
	hcr.lock.RLock()
	defer hcr.lock.RUnlock()

	v, ok := hcr.channels[chainID]
	if !ok {
		return nil
	}
	return v
}

// GetEarnings returns all channels earnings for given identity
func (hcr *HermesChannelRepository) GetEarnings(chainID int64, id identity.Identity) event.Earnings {
	hcr.lock.RLock()
	defer hcr.lock.RUnlock()

	return hcr.sumChannels(chainID, id)
}

func (hcr *HermesChannelRepository) sumChannels(chainID int64, id identity.Identity) event.Earnings {
	var lifetimeBalance = new(big.Int)
	var unsettledBalance = new(big.Int)
	v, ok := hcr.channels[chainID]
	if !ok {
		return event.Earnings{
			LifetimeBalance:  new(big.Int),
			UnsettledBalance: new(big.Int),
		}
	}

	for _, channel := range v {
		if channel.Identity == id {
			lifetimeBalance = new(big.Int).Add(lifetimeBalance, channel.LifetimeBalance())
			unsettledBalance = new(big.Int).Add(unsettledBalance, channel.UnsettledBalance())
		}
	}

	return event.Earnings{
		LifetimeBalance:  lifetimeBalance,
		UnsettledBalance: unsettledBalance,
	}
}

// Subscribe subscribes to the appropriate events.
func (hcr *HermesChannelRepository) Subscribe(bus eventbus.Subscriber) error {
	err := bus.SubscribeAsync(nodevent.AppTopicNode, hcr.handleNodeStart)
	if err != nil {
		return fmt.Errorf("could not subscribe to node status event: %w", err)
	}
	err = bus.SubscribeAsync(pinge.AppTopicHermesPromise, hcr.handleHermesPromiseReceived)
	if err != nil {
		return fmt.Errorf("could not subscribe to AppTopicHermesPromise event: %w", err)
	}
	return nil
}

func (hcr *HermesChannelRepository) handleHermesPromiseReceived(payload pinge.AppEventHermesPromise) {
	channelID, err := crypto.GenerateProviderChannelID(payload.ProviderID.Address, payload.HermesID.Hex())
	if err != nil {
		log.Err(err).Msg("could not generate provider channel id")
		return
	}

	promise, err := hcr.promiseProvider.Get(payload.Promise.ChainID, channelID)
	if err != nil {
		log.Err(err).Msgf("could not get hermes promise for provider %v, hermes %v", payload.ProviderID, payload.HermesID.Hex())
		return
	}

	err = hcr.updateChannelWithLatestPromise(payload.Promise.ChainID, promise.ChannelID, payload.ProviderID, payload.HermesID, promise)
	if err != nil {
		log.Err(err).Msg("could not update channel state with latest hermes promise")
	}
}

func (hcr *HermesChannelRepository) handleNodeStart(payload nodevent.Payload) {
	if payload.Status != nodevent.StatusStarted {
		return
	}
	hcr.fetchKnownChannels(config.GetInt64(config.FlagChainID))
}

func (hcr *HermesChannelRepository) fetchKnownChannels(chainID int64) {
	hcr.lock.Lock()
	defer hcr.lock.Unlock()

	promises, err := hcr.promiseProvider.List(HermesPromiseFilter{
		ChainID: chainID,
	})
	if err != nil {
		log.Error().Err(err).Msg("could not load initial earnings state")
		return
	}

	for _, promise := range promises {
		if _, err := hcr.fetchChannel(chainID, promise.ChannelID, promise.Identity, promise.HermesID, promise); err != nil {
			log.Error().Err(err).Msg("could not load initial earnings state")
		}
	}
}

func (hcr *HermesChannelRepository) fetchChannel(chainID int64, channelID string, id identity.Identity, hermesID common.Address, promise HermesPromise) (HermesChannel, error) {
	// TODO Should call GetProviderChannelByID() but can't pass pending=false
	// This will get retried so we do not need to explicitly retry
	// TODO: maybe add a sane limit of retries
	channel, err := hcr.channelProvider.GetProviderChannel(chainID, hermesID, id.ToCommonAddress(), true)
	if err != nil {
		return HermesChannel{}, fmt.Errorf("could not get provider channel for %v, hermes %v: %w", id, hermesID.Hex(), err)
	}

	hermesChannel := NewHermesChannel(channelID, id, hermesID, channel, promise)

	benef, err := hcr.bprovider.GetBeneficiary(id.ToCommonAddress())
	if err != nil {
		return HermesChannel{}, fmt.Errorf("could not get provider beneficiary for %v, hermes %v: %w", id, hermesID.Hex(), err)
	}

	hermesChannel.Beneficiary = benef

	hcr.updateChannel(chainID, hermesChannel)

	return hermesChannel, nil
}

func (hcr *HermesChannelRepository) updateChannelWithLatestPromise(chainID int64, channelID string, id identity.Identity, hermesID common.Address, promise HermesPromise) error {
	gotten, ok := hcr.Get(chainID, id, hermesID)
	if !ok {
		// this actually performs the update, so no need to do anything
		_, err := hcr.fetchChannel(chainID, channelID, id, hermesID, promise)
		return err
	}

	hermesChannel := NewHermesChannel(channelID, id, hermesID, gotten.Channel, promise)
	hcr.updateChannel(chainID, hermesChannel)
	return nil
}

func (hcr *HermesChannelRepository) updateChannel(chainID int64, new HermesChannel) {
	earningsOld := hcr.sumChannels(chainID, new.Identity)

	updated := false

	v := hcr.channels[chainID]
	for i, channel := range v {
		if channel.Identity == new.Identity && channel.HermesID == new.HermesID {
			updated = true
			hcr.channels[chainID][i] = new
			break
		}
	}
	res := append(hcr.channels[chainID], new)
	if !updated {
		hcr.channels[chainID] = res
	}

	log.Info().Msgf(
		"Loaded state for provider %q, hermesID %q: balance %v, available balance %v, unsettled balance %v",
		new.Identity,
		new.HermesID.Hex(),
		new.balance(),
		new.availableBalance(),
		new.UnsettledBalance(),
	)

	earningsNew := hcr.sumChannels(chainID, new.Identity)
	go hcr.publisher.Publish(event.AppTopicEarningsChanged, event.AppEventEarningsChanged{
		Identity: new.Identity,
		Previous: earningsOld,
		Current:  earningsNew,
	})
}
