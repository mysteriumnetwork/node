/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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

package migration

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/rs/zerolog/log"
)

const oldBalanceMigrationMinimumMyst = 0.1

var openChannelTimeout = time.Hour

// HermesMigrator migrate identity from old hermes to new.
// It opens a new channel for new Hermes and sends all MYST to a new payment channel.
type HermesMigrator struct {
	transactor          *registry.Transactor
	addressProvider     registry.AddressProvider
	hps                 pingpong.HermesPromiseSettler
	hermesURLGetter     *pingpong.HermesURLGetter
	hermesCallerFactory pingpong.HermesCallerFactory
	registry            registry.IdentityRegistry
	cbt                 *pingpong.ConsumerBalanceTracker
}

// NewHermesMigrator create new HermesMigrator
func NewHermesMigrator(
	transactor *registry.Transactor,
	addressProvider registry.AddressProvider,
	hermesURLGetter *pingpong.HermesURLGetter,
	hermesCallerFactory pingpong.HermesCallerFactory,
	hps pingpong.HermesPromiseSettler,
	registry registry.IdentityRegistry,
	cbt *pingpong.ConsumerBalanceTracker,
) *HermesMigrator {
	return &HermesMigrator{
		transactor:          transactor,
		addressProvider:     addressProvider,
		hermesURLGetter:     hermesURLGetter,
		hermesCallerFactory: hermesCallerFactory,
		hps:                 hps,
		registry:            registry,
		cbt:                 cbt,
	}
}

// HermesClient responses for receiving info from Hermes
type HermesClient interface {
	GetConsumerData(chainID int64, id string) (pingpong.HermesUserInfo, error)
}

// Start begins migration from old hermes to new
func (m *HermesMigrator) Start(id string) error {
	chainID := config.GetInt64(config.FlagChainID)

	registered, err := m.isRegistered(chainID, id)
	if err != nil {
		return fmt.Errorf("could not get identity register status: %w", err)
	} else if !registered {
		return errors.New("identity is unregistered")
	}

	activeHermes, err := m.addressProvider.GetActiveHermes(chainID)
	if err != nil {
		return fmt.Errorf("could not get hermes address: %w", err)
	}
	knownHermeses, err := m.addressProvider.GetKnownHermeses(chainID)
	if err != nil {
		return fmt.Errorf("could not get hermes address: %w", err)
	}
	oldHermeses := getOldHermeses(knownHermeses, activeHermes)
	if len(oldHermeses) != 1 {
		return nil
	}
	oldHermes := oldHermeses[0]

	registryAddress, err := m.addressProvider.GetRegistryAddress(chainID)
	if err != nil {
		return fmt.Errorf("could not get registry address: %w", err)
	}
	oldBalance, err := m.getBalance(chainID, oldHermes.Hex(), id)
	if err != nil {
		return fmt.Errorf("error during getting balance: %w", err)
	}

	if err = m.openChannel(id, err, chainID, activeHermes, registryAddress); err != nil {
		return fmt.Errorf("open channel error: %w", err)
	}

	channelImpl, err := m.addressProvider.GetActiveChannelImplementation(chainID)
	if err != nil {
		return fmt.Errorf("error during getting channel implementation: %w", err)
	}
	newChannel, err := crypto.GenerateChannelAddress(id, activeHermes.Hex(), registryAddress.Hex(), channelImpl.Hex())
	if err != nil {
		return fmt.Errorf("generate channel address erro: %w", err)
	}

	providerId := identity.FromAddress(id)

	if crypto.FloatToBigMyst(oldBalanceMigrationMinimumMyst).Cmp(oldBalance) > 0 {
		// If not enough balance we should still check that latest withdrawal succeeded
		amountToWithdraw, chid, err := m.hps.CheckLatestWithdrawal(chainID, identity.FromAddress(id), oldHermes)
		if err != nil {
			if !errors.Is(err, pingpong.ErrNotFound) {
				return fmt.Errorf("failed to check latest withdrawal status: %w", err)
			}
			log.Warn().Err(err).Msg("No promise saved")
		} else if amountToWithdraw != nil && amountToWithdraw.Cmp(big.NewInt(0)) == 1 {
			log.Debug().Msgf("Found withdrawal which is not settled, will retry to withdraw")
			return m.hps.RetryWithdrawLatest(chainID, amountToWithdraw, chid, common.HexToAddress(newChannel), providerId)
		}
		log.Debug().Msgf("Not enough balance for migration (id: %s, old balance: %.2f)", id, crypto.BigMystToFloat(oldBalance))
		return nil
	}

	log.Debug().Msgf("Send transaction. Old Hermes: %s, new Hermes %s (channel: %s)", oldHermes, activeHermes, newChannel)

	if err := m.hps.Withdraw(chainID, chainID, providerId, oldHermes, common.HexToAddress(newChannel), nil); err != nil {
		return err
	}

	m.cbt.ForceBalanceUpdateCached(chainID, providerId)

	return nil
}

func (m *HermesMigrator) openChannel(id string, err error, chainID int64, activeHermes common.Address, registryAddress common.Address) error {
	statusResponse, err := m.transactor.ChannelStatus(chainID, id, activeHermes.Hex(), registryAddress.Hex())
	if err != nil {
		return fmt.Errorf("channel status error: %w", err)
	}

	log.Debug().Msgf("Channel status: %s", statusResponse.Status)
	if statusResponse.Status == registry.ChannelStatusNotFound {
		if err := m.transactor.OpenChannel(chainID, id, activeHermes.Hex(), registryAddress.Hex()); err != nil {
			return fmt.Errorf("open new channel error: %w", err)
		}
		err = m.waitForChannelOpened(chainID, common.HexToAddress(id), activeHermes, registryAddress, openChannelTimeout)
	} else if statusResponse.Status == registry.ChannelStatusInProgress {
		err = m.waitForChannelOpened(chainID, common.HexToAddress(id), activeHermes, registryAddress, openChannelTimeout)
	}
	if err != nil {
		return fmt.Errorf("error during waiting for channel opening: %w", err)
	}

	return nil
}

// IsMigrationRequired check whether migration required
func (m *HermesMigrator) IsMigrationRequired(id string) (bool, error) {
	chainID := config.GetInt64(config.FlagChainID)

	registered, err := m.isRegistered(chainID, id)
	if err != nil {
		return false, fmt.Errorf("could not get identity register status: %w", err)
	} else if !registered {
		return false, nil
	}

	activeHermes, err := m.addressProvider.GetActiveHermes(chainID)
	if err != nil {
		return false, fmt.Errorf("could not get hermes address: %w", err)
	}

	knownHermeses, err := m.addressProvider.GetKnownHermeses(chainID)
	if err != nil {
		return false, fmt.Errorf("could not get hermes address: %w", err)
	}
	oldHermeses := getOldHermeses(knownHermeses, activeHermes)
	if len(oldHermeses) != 1 {
		return false, nil
	}
	oldHermes := oldHermeses[0]

	registryAddress, err := m.addressProvider.GetRegistryAddress(chainID)
	if err != nil {
		return false, fmt.Errorf("could not get registry address: %w", err)
	}

	statusResponse, err := m.transactor.ChannelStatus(chainID, id, activeHermes.Hex(), registryAddress.Hex())
	if err != nil {
		return false, err
	}
	if statusResponse.Status == registry.ChannelStatusNotFound || statusResponse.Status == registry.ChannelStatusInProgress {
		return true, nil
	}

	oldBalance, err := m.getBalance(chainID, oldHermes.Hex(), id)
	if err != nil {
		return false, fmt.Errorf("error during getting balance: %w", err)
	}
	newBalance, err := m.getBalance(chainID, activeHermes.Hex(), id)
	if err != nil {
		return false, fmt.Errorf("error during getting balance: %w", err)
	}

	amountToWithdraw, _, err := m.hps.CheckLatestWithdrawal(chainID, identity.FromAddress(id), oldHermes)
	if err != nil {
		if !errors.Is(err, pingpong.ErrNotFound) {
			return false, fmt.Errorf("failed to check latest withdrawal status: %w", err)
		}
		log.Warn().Err(err).Msg("No promise saved")
	} else if amountToWithdraw != nil && amountToWithdraw.Cmp(big.NewInt(0)) == 1 {
		return true, nil
	}

	return crypto.FloatToBigMyst(oldBalanceMigrationMinimumMyst).Cmp(oldBalance) < 0 && newBalance.Cmp(new(big.Int)) == 0, nil
}

func (m *HermesMigrator) waitForChannelOpened(chainID int64, id, hermesID, registryAddress common.Address, timeout time.Duration) error {
	log.Debug().Msg("Hermes migration: opening new channel")
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return fmt.Errorf("channel opening timeout for id: %s", id)
		case <-time.After(5 * time.Second):
			statusResponse, err := m.transactor.ChannelStatus(chainID, id.Hex(), hermesID.Hex(), registryAddress.Hex())
			if err != nil {
				log.Debug().Msgf("Hermes migration error: open channel failed %s", err.Error())
				return err
			} else if statusResponse.Status == registry.ChannelStatusFail || statusResponse.Status == registry.ChannelStatusOpen {
				log.Debug().Msg("Hermes migration: new channel opened successfully")
				return nil
			}
		}
	}
}

func (m *HermesMigrator) getHermesCaller(chainID int64, hermesID string) (HermesClient, error) {
	addr, err := m.hermesURLGetter.GetHermesURL(chainID, common.HexToAddress(hermesID))
	if err != nil {
		return nil, err
	}

	return m.hermesCallerFactory(addr), nil
}

// getBalance gets the current balance for given identity
func (m *HermesMigrator) getBalance(chainID int64, hermesID, id string) (*big.Int, error) {
	c, err := m.getHermesCaller(chainID, hermesID)
	if err != nil {
		return nil, err
	}

	data, err := c.GetConsumerData(chainID, id)
	if err != nil {
		if errors.Is(err, pingpong.ErrHermesNotFound) {
			return new(big.Int), nil
		}
		return nil, err
	}

	return data.Balance, nil
}

func getOldHermeses(knownHermeses []common.Address, activeHermes common.Address) []common.Address {
	oldHermeses := []common.Address{}
	for _, address := range knownHermeses {
		if address.Hex() != activeHermes.Hex() {
			oldHermeses = append(oldHermeses, address)
		}
	}

	return oldHermeses
}

func (m *HermesMigrator) isRegistered(chainID int64, id string) (bool, error) {
	status, err := m.registry.GetRegistrationStatus(chainID, identity.FromAddress(id))
	if err != nil {
		return false, err
	}
	return status == registry.Registered, nil
}
