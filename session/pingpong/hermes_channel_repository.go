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
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	nodeEvent "github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	pingEvent "github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/payments/client"
	"github.com/mysteriumnetwork/payments/crypto"
)

type promiseProvider interface {
	Get(chainID int64, channelID string) (HermesPromise, error)
	List(filter HermesPromiseFilter) ([]HermesPromise, error)
	Store(promise HermesPromise) error
}

type channelProvider interface {
	GetProviderChannel(chainID int64, hermesAddress common.Address, addressToCheck common.Address, pending bool) (client.ProviderChannel, error)
}

type hermesCaller interface {
	GetProviderData(chainID int64, id string) (HermesUserInfo, error)
	RefreshLatestProviderPromise(chainID int64, id string, hashlock, recoveryData []byte, signer identity.Signer) (crypto.Promise, error)
	RevealR(r string, provider string, agreementID *big.Int) error
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
	addressProvider addressProvider
	hermesCaller    hermesCaller
	encryption      encryption
	bprovider       beneficiaryProvider
	lock            sync.RWMutex
	signer          identity.SignerFactory
}

// NewHermesChannelRepository returns a new instance of HermesChannelRepository.
func NewHermesChannelRepository(promiseProvider promiseProvider, channelProvider channelProvider, publisher eventbus.Publisher, bprovider beneficiaryProvider, hermesCaller hermesCaller, addressProvider addressProvider, signer identity.SignerFactory, encryption encryption) *HermesChannelRepository {
	return &HermesChannelRepository{
		promiseProvider: promiseProvider,
		channelProvider: channelProvider,
		publisher:       publisher,
		bprovider:       bprovider,
		hermesCaller:    hermesCaller,
		addressProvider: addressProvider,
		channels:        make(map[int64][]HermesChannel, 0),
		signer:          signer,
		encryption:      encryption,
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

			// return a copy!!!
			return channel.Copy(), true
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

	// make a copy of array, so it could be used in other goroutines
	channelsCopy := make([]HermesChannel, 0)
	for _, val := range v {
		channelsCopy = append(channelsCopy, val.Copy())
	}
	return channelsCopy
}

// GetEarnings returns all channels earnings for given identity combined from all hermeses possible
func (hcr *HermesChannelRepository) GetEarnings(chainID int64, id identity.Identity) pingEvent.Earnings {
	hcr.lock.RLock()
	defer hcr.lock.RUnlock()

	return hcr.sumChannels(chainID, id)
}

// GetEarningsDetailed returns earnings in a detailed format grouping them by hermes ID but also providing totals.
func (hcr *HermesChannelRepository) GetEarningsDetailed(chainID int64, id identity.Identity) *pingEvent.EarningsDetailed {
	hcr.lock.RLock()
	defer hcr.lock.RUnlock()

	return hcr.sumChannelsDetailed(chainID, id)
}

func (hcr *HermesChannelRepository) sumChannelsDetailed(chainID int64, id identity.Identity) *pingEvent.EarningsDetailed {
	result := &pingEvent.EarningsDetailed{
		Total: pingEvent.Earnings{
			LifetimeBalance:  new(big.Int),
			UnsettledBalance: new(big.Int),
		},
		PerHermes: make(map[common.Address]pingEvent.Earnings),
	}

	v, ok := hcr.channels[chainID]
	if !ok {
		return result
	}

	add := func(current pingEvent.Earnings, channel HermesChannel) pingEvent.Earnings {
		life := new(big.Int).Add(current.LifetimeBalance, channel.LifetimeBalance())
		unset := new(big.Int).Add(current.UnsettledBalance, channel.UnsettledBalance())

		// Save total globally per all hermeses
		current.LifetimeBalance = life
		current.UnsettledBalance = unset

		return current
	}

	for _, channel := range v {
		if channel.Identity != id {
			continue
		}
		result.Total = add(result.Total, channel)

		// Save total for a single hermes
		got, ok := result.PerHermes[channel.HermesID]
		if !ok {
			got = pingEvent.Earnings{
				LifetimeBalance:  new(big.Int),
				UnsettledBalance: new(big.Int),
			}
		}

		result.PerHermes[channel.HermesID] = add(got, channel)
	}

	return result
}

func (hcr *HermesChannelRepository) sumChannels(chainID int64, id identity.Identity) pingEvent.Earnings {
	var lifetimeBalance = new(big.Int)
	var unsettledBalance = new(big.Int)
	v, ok := hcr.channels[chainID]
	if !ok {
		return pingEvent.Earnings{
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

	return pingEvent.Earnings{
		LifetimeBalance:  lifetimeBalance,
		UnsettledBalance: unsettledBalance,
	}
}

// Subscribe subscribes to the appropriate events.
func (hcr *HermesChannelRepository) Subscribe(bus eventbus.Subscriber) error {
	err := bus.SubscribeAsync(nodeEvent.AppTopicNode, hcr.handleNodeStart)
	if err != nil {
		return fmt.Errorf("could not subscribe to node status event: %w", err)
	}
	err = bus.SubscribeAsync(pingEvent.AppTopicHermesPromise, hcr.handleHermesPromiseReceived)
	if err != nil {
		return fmt.Errorf("could not subscribe to AppTopicHermesPromise event: %w", err)
	}
	err = bus.SubscribeAsync(identity.AppTopicIdentityUnlock, hcr.handleIdentityUnlock)
	if err != nil {
		return fmt.Errorf("could not subscribe to AppTopicIdentityUnlock event: %w", err)
	}
	return nil
}

func (hcr *HermesChannelRepository) handleHermesPromiseReceived(payload pingEvent.AppEventHermesPromise) {
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

	// use parameter "protectChannels" to protect channels on update
	err = hcr.updateChannelWithLatestPromise(payload.Promise.ChainID, promise.ChannelID, payload.ProviderID, payload.HermesID, promise, true)
	if err != nil {
		log.Err(err).Msg("could not update channel state with latest hermes promise")
	}
}

func (hcr *HermesChannelRepository) handleNodeStart(payload nodeEvent.Payload) {
	if payload.Status != nodeEvent.StatusStarted {
		return
	}
	hcr.fetchKnownChannels(config.GetInt64(config.FlagChainID))
}

func (hcr *HermesChannelRepository) handleIdentityUnlock(payload identity.AppEventIdentityUnlock) {
	hermes, err := hcr.addressProvider.GetActiveHermes(payload.ChainID)
	if err != nil {
		log.Err(err).Msg("failed to get active Hermes")
		return
	}
	hermesChannel, exists := hcr.Get(payload.ChainID, payload.ID, hermes)
	if exists {
		return
	}

	unsettledBalance := hermesChannel.UnsettledBalance()
	if unsettledBalance.Cmp(big.NewInt(0)) != 0 {
		return
	}

	data, err := hcr.hermesCaller.GetProviderData(payload.ChainID, payload.ID.Address)
	if err != nil {
		log.Err(err).Msg("failed to get provider data")
		return
	}

	//skip refresh if promise has been revealed or we know r
	if data.LatestPromise.Hashlock != "" {
		hermesPromise, err := hcr.promiseProvider.Get(payload.ChainID, data.ChannelID)
		if err == nil && strings.EqualFold(data.LatestPromise.Hashlock, fmt.Sprintf("0x%s", common.Bytes2Hex(hermesPromise.Promise.Hashlock))) {
			err = hcr.revealR(hermesPromise)
			if err == nil {
				return
			}
			log.Error().Err(err).Msgf("failed to reveal R on identity unlock")
		}
	}

	if data.LatestPromise.Amount != nil && data.LatestPromise.Amount.Cmp(big.NewInt(0)) != 0 {
		R, err := crypto.GenerateR()
		if err != nil {
			log.Err(err).Msg("failed to generate R")
			return
		}
		hashlock := ethcrypto.Keccak256(R)
		details := rRecoveryDetails{
			R:           hex.EncodeToString(R),
			AgreementID: big.NewInt(0),
		}

		bytes, err := json.Marshal(details)
		if err != nil {
			log.Err(err).Msgf("could not marshal R recovery details")
			return
		}

		encrypted, err := hcr.encryption.Encrypt(payload.ID.ToCommonAddress(), bytes)
		if err != nil {
			log.Err(err).Msgf("could not encrypt R")
			return
		}
		signer := hcr.signer(payload.ID)
		promise, err := hcr.hermesCaller.RefreshLatestProviderPromise(config.GetInt64(config.FlagChainID), payload.ID.Address, hashlock, encrypted, signer)
		if err != nil {
			log.Err(err).Msgf("failed to refresh promise")
			return
		}
		hermesPromise := HermesPromise{
			R:         hex.EncodeToString(R),
			ChannelID: data.ChannelID,
			Identity:  identity.FromAddress(data.Identity),
			HermesID:  hermes,
			Promise:   promise,
			Revealed:  false,
		}

		err = hcr.promiseProvider.Store(hermesPromise)
		if err != nil {
			log.Err(err).Msg("could not store hermes promise")
			return
		}
		hcr.publisher.Publish(pingEvent.AppTopicHermesPromise, pingEvent.AppEventHermesPromise{
			Promise:    promise,
			HermesID:   hermes,
			ProviderID: identity.FromAddress(data.Identity),
		})

		err = hcr.revealR(hermesPromise)
		if err != nil {
			log.Err(err).Msgf("failed to reveal R after promise refresh")
		}
		log.Debug().Bool("saved", err == nil).Msg("refreshed promise")
	}
}

func (hcr *HermesChannelRepository) revealR(hermesPromise HermesPromise) error {
	if hermesPromise.Revealed {
		return nil
	}

	err := hcr.hermesCaller.RevealR(hermesPromise.R, hermesPromise.Identity.Address, hermesPromise.AgreementID)
	if err != nil {
		return fmt.Errorf("could not reveal R: %w", err)
	}

	hermesPromise.Revealed = true
	err = hcr.promiseProvider.Store(hermesPromise)
	if err != nil && !errors.Is(err, ErrAttemptToOverwrite) {
		return fmt.Errorf("could not store hermes promise: %w", err)
	}

	return nil
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
		gen, err := crypto.GenerateProviderChannelID(promise.Identity.Address, promise.HermesID.Hex())
		if err != nil {
			log.Err(err).Msg("could not generate a provider channel address")
			continue
		}
		if strings.ToLower(gen) != strings.ToLower(promise.ChannelID) {
			log.Debug().Fields(map[string]interface{}{
				"identity":           promise.Identity.Address,
				"expected_channelID": gen,
				"got_channelID":      promise.ChannelID,
				"hermes":             promise.HermesID.Hex(),
			}).Msg("promise channel ID did not match provider channel ID, skipping")
			continue
		}

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

	benef, err := hcr.bprovider.GetBeneficiary(id.ToCommonAddress())
	if err != nil {
		return HermesChannel{}, fmt.Errorf("could not get provider beneficiary for %v, hermes %v: %w", id, hermesID.Hex(), err)
	}
	hermesChannel := NewHermesChannel(channelID, id, hermesID, channel, promise, benef).
		Copy()

	hcr.updateChannel(chainID, hermesChannel)

	return hermesChannel, nil
}

func (hcr *HermesChannelRepository) updateChannelWithLatestPromise(chainID int64, channelID string, id identity.Identity, hermesID common.Address, promise HermesPromise, protectChannels bool) error {
	gotten, ok := hcr.Get(chainID, id, hermesID)
	if !ok {
		// this actually performs the update, so no need to do anything
		_, err := hcr.fetchChannel(chainID, channelID, id, hermesID, promise)
		return err
	}

	hermesChannel := NewHermesChannel(channelID, id, hermesID, gotten.Channel, promise, gotten.Beneficiary).
		Copy()

	// protect hcr.channels: handleHermesPromiseReceived -> updateChannelWithLatestPromise -> updateChannel
	if protectChannels {
		hcr.lock.Lock()
		defer hcr.lock.Unlock()
	}
	hcr.updateChannel(chainID, hermesChannel)

	return nil
}

func (hcr *HermesChannelRepository) updateChannel(chainID int64, new HermesChannel) {
	earningsOld := hcr.sumChannelsDetailed(chainID, new.Identity)

	updated := false

	v := hcr.channels[chainID]
	for i, channel := range v {
		if channel.Identity == new.Identity && channel.HermesID == new.HermesID {
			updated = true
			// rewrites element in array. to prevent data race on array elements - use its deep-copy in other goroutines
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

	earningsNew := hcr.sumChannelsDetailed(chainID, new.Identity)
	go hcr.publisher.Publish(pingEvent.AppTopicEarningsChanged, pingEvent.AppEventEarningsChanged{
		Identity: new.Identity,
		Previous: *earningsOld,
		Current:  *earningsNew,
	})
}
