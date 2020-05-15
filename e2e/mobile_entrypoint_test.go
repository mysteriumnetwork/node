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

	"github.com/mysteriumnetwork/node/mobile/mysterium"
	"github.com/stretchr/testify/require"
)

func TestMobileNodeConsumer(t *testing.T) {
	dir, err := ioutil.TempDir("", "mobileEntryPoint")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	options := &mysterium.MobileNodeOptions{
		Testnet:                         true,
		ExperimentNATPunching:           true,
		MysteriumAPIAddress:             "http://mysterium-api:8001/v1",
		BrokerAddress:                   "broker",
		EtherClientRPC:                  "ws://ganache:8545",
		FeedbackURL:                     "TODO",
		QualityOracleURL:                "http://morqa:8085/api/v1",
		IPDetectorURL:                   "http://ipify:3000/?format=json",
		LocationDetectorURL:             "https://testnet-location.mysterium.network/api/v1/location",
		TransactorEndpointAddress:       "http://transactor:8888/api/v1",
		TransactorRegistryAddress:       "0xbe180c8CA53F280C7BE8669596fF7939d933AA10",
		TransactorChannelImplementation: "0x599d43715DF3070f83355D9D90AE62c159E62A75",
		AccountantEndpointAddress:       "http://accountant:8889/api/v1",
		AccountantID:                    "0x7621a5E6EC206309f8E703A653f03F7C8a3097a8",
		MystSCAddress:                   "0x4D1d104AbD4F4351a0c51bE1e9CA0750BbCa1665",
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

		err = node.RegisterIdentity(&mysterium.RegisterIdentityRequest{
			IdentityAddress: identity.IdentityAddress,
			Fee:             10000000,
		})
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			identity, err := node.GetIdentity(&mysterium.GetIdentityRequest{})
			require.NoError(t, err)
			return identity.RegistrationStatus == "RegisteredConsumer"
		}, 15*time.Second, 1*time.Second)
	})

	t.Run("Test balance", func(t *testing.T) {
		identity, err := node.GetIdentity(&mysterium.GetIdentityRequest{})
		require.NoError(t, err)

		balance, err := node.GetBalance(&mysterium.GetBalanceRequest{IdentityAddress: identity.IdentityAddress})
		require.NoError(t, err)
		require.Equal(t, int64(690000000), balance.Balance)
	})

	t.Run("Test shutdown", func(t *testing.T) {
		err := node.Shutdown()
		require.NoError(t, err)
	})
}
