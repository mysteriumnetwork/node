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

package e2e

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/mobile/mysterium"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/stretchr/testify/require"
)

func TestMobileNodeConsumer(t *testing.T) {
	dir, err := ioutil.TempDir("", "mobileEntryPoint")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	options := &mysterium.MobileNodeOptions{
		Betanet:                         true,
		ExperimentNATPunching:           true,
		MysteriumAPIAddress:             "http://mysterium-api:8001/v1",
		BrokerAddress:                   "broker",
		EtherClientRPC:                  "ws://ganache:8545",
		FeedbackURL:                     "TODO",
		QualityOracleURL:                "http://morqa:8085/api/v1",
		IPDetectorURL:                   "http://ipify:3000/?format=json",
		LocationDetectorURL:             "https://testnet-location.mysterium.network/api/v1/location",
		TransactorEndpointAddress:       "http://transactor:8888/api/v1",
		TransactorRegistryAddress:       registryAddress,
		TransactorChannelImplementation: channelImplementation,
		HermesEndpointAddress:           "http://hermes:8889/api/v1",
		HermesID:                        hermesID,
		MystSCAddress:                   mystAddress,
	}

	node, err := mysterium.NewNode(dir, options)
	require.NoError(t, err)
	require.NotNil(t, node)

	t.Run("Test status", func(t *testing.T) {
		status := node.GetStatus()
		require.Equal(t, "NotConnected", status.State)
		require.Equal(t, "", status.ProviderID)
		require.Equal(t, "", status.ServiceType)
	})

	t.Run("Test identity registration", func(t *testing.T) {
		identity, err := node.GetIdentity(&mysterium.GetIdentityRequest{})
		require.NoError(t, err)

		require.NotNil(t, identity)
		require.Equal(t, "Unregistered", identity.RegistrationStatus)

		topUpConsumer(t, identity.IdentityAddress, common.HexToAddress(options.HermesID), registrationFee)

		err = node.RegisterIdentity(&mysterium.RegisterIdentityRequest{
			IdentityAddress: identity.IdentityAddress,
		})
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			identity, err := node.GetIdentity(&mysterium.GetIdentityRequest{})
			require.NoError(t, err)
			return identity.RegistrationStatus == "Registered"
		}, 15*time.Second, 1*time.Second)
	})

	t.Run("Test balance", func(t *testing.T) {
		identity, err := node.GetIdentity(&mysterium.GetIdentityRequest{})
		require.NoError(t, err)

		balance, err := node.GetBalance(&mysterium.GetBalanceRequest{IdentityAddress: identity.IdentityAddress})
		require.NoError(t, err)
		require.Equal(t, crypto.BigMystToFloat(balanceAfterRegistration), balance.Balance)
	})

	t.Run("Test shutdown", func(t *testing.T) {
		err := node.Shutdown()
		require.NoError(t, err)
	})
}
