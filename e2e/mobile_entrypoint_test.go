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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/mobile/mysterium"
	"github.com/mysteriumnetwork/payments/crypto"
)

func TestMobileNodeConsumer(t *testing.T) {
	dir, err := os.MkdirTemp("", "mobileEntryPoint")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	err = os.Mkdir(filepath.Join(dir, ".mysterium"), 0777)
	require.NoError(t, err)

	options := &mysterium.MobileNodeOptions{
		Network:                        string(config.Mainnet),
		DiscoveryAddress:               "http://discovery:8080/api/v4",
		BrokerAddresses:                []string{"broker"},
		EtherClientRPCL1:               []string{"http://ganache:8545"},
		EtherClientRPCL2:               []string{"ws://ganache2:8545"},
		FeedbackURL:                    "TODO",
		QualityOracleURL:               "http://morqa:8085/api/v3",
		IPDetectorURL:                  "http://ipify:3000/?format=json",
		LocationDetectorURL:            "https://location.mysterium.network/api/v1/location",
		TransactorEndpointAddress:      "http://transactor:8888/api/v1",
		HermesEndpointAddress:          "http://hermes:8889/api/v1",
		ActiveChainID:                  80001,
		Chain1ID:                       5,
		Chain2ID:                       80001,
		MystSCAddress:                  "0xaa9c4e723609cb913430143fbc86d3cbe7adca21",
		RegistrySCAddress:              "0x427c2bad22335710aec5e477f3e3adcd313a9bcb",
		HermesSCAddress:                "0xd68defb97d0765741f8ecf179df2f9564e1466a3",
		ChannelImplementationSCAddress: "0x599d43715df3070f83355d9d90ae62c159e62a75",
	}

	node, err := mysterium.NewNode(dir, options)
	require.NoError(t, err)
	require.NotNil(t, node)

	t.Run("Test status", func(t *testing.T) {
		resp, err := node.GetStatus()
		require.NoError(t, err)

		var status mysterium.GetStatusResponse
		err = json.Unmarshal(resp, &status)

		require.NoError(t, err)

		require.Equal(t, "NotConnected", status.State)
		require.Equal(t, "", status.Proposal.ProviderID)
		require.Equal(t, "", status.Proposal.ServiceType)
	})

	t.Run("Test identity registration", func(t *testing.T) {
		identity, err := node.GetIdentity(&mysterium.GetIdentityRequest{})
		require.NoError(t, err)

		require.NotNil(t, identity)
		require.Equal(t, "Unregistered", identity.RegistrationStatus)

		topUpConsumer(t, identity.IdentityAddress, common.HexToAddress(hermesID), registrationFee)

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

		assert.Eventually(t, func() bool {
			balance, err := node.GetBalance(&mysterium.GetBalanceRequest{IdentityAddress: identity.IdentityAddress})
			require.NoError(t, err)
			return crypto.BigMystToFloat(balanceAfterRegistration) == balance.Balance
		}, time.Second*5, 100*time.Millisecond)
	})

	t.Run("Test identity export", func(t *testing.T) {
		identity, err := node.GetIdentity(&mysterium.GetIdentityRequest{})
		require.NoError(t, err)
		// without '0x' prefix
		hexAddress := strings.ToLower(identity.IdentityAddress[2:])

		exportBytes, err := node.ExportIdentity(identity.IdentityAddress, "secret_pass")
		require.NoError(t, err)

		var ks identityKeystore
		err = json.Unmarshal(exportBytes, &ks)
		require.NoError(t, err)
		require.Equal(t, ks.Address, hexAddress)
		require.NotEmpty(t, ks.Version)
		require.NotEmpty(t, ks.ID)
		require.NotEmpty(t, ks.Crypto)
	})

	t.Run("Test identity import", func(t *testing.T) {
		keystoreString := "{\"address\":\"2574e9053c104f5e6012cbb0aa457318339d8a7f\",\"crypto\":{\"cipher\":\"aes-128-ctr\",\"ciphertext\":\"772b3df26635c50fccf26350c6530c4216e2d78b4836105475f2876dc0704810\",\"cipherparams\":{\"iv\":\"1b96fb8b5614f5b46f1e1e0327f370ed\"},\"kdf\":\"scrypt\",\"kdfparams\":{\"dklen\":32,\"n\":4096,\"p\":6,\"r\":8,\"salt\":\"6978a44ba80d588aacf497d2b042948bdbf74aefa22b715ab863647511236f17\"},\"mac\":\"77b896027172c9dc68d64f15d6450492bd92a57b994734fd147769a580e02ef6\"},\"id\":\"d18381e4-2011-48c7-97cf-84ccc3882c87\",\"version\":3}"
		keystorePass := "fhHGF12G2g"

		address, err := node.ImportIdentity([]byte(keystoreString), keystorePass)
		require.NoError(t, err)
		require.NotEmpty(t, address)

		identity, err := node.GetIdentity(&mysterium.GetIdentityRequest{Address: address})
		require.NoError(t, err)
		require.Equal(t, address, identity.IdentityAddress)
		require.NotEmpty(t, identity.ChannelAddress)
		require.Equal(t, "Unregistered", identity.RegistrationStatus)
	})

	t.Run("Test resident country", func(t *testing.T) {
		// given
		identity, err := node.GetIdentity(&mysterium.GetIdentityRequest{})
		require.NoError(t, err)

		// when
		err = node.UpdateResidentCountry(&mysterium.ResidentCountryUpdateRequest{IdentityAddress: identity.IdentityAddress, Country: "AU"})
		require.NoError(t, err)

		// then
		require.Equal(t, "AU", node.ResidentCountry(), "default country should be set")

		// and
		err = node.UpdateResidentCountry(&mysterium.ResidentCountryUpdateRequest{IdentityAddress: identity.IdentityAddress})
		require.Error(t, err, "country is required")
		err = node.UpdateResidentCountry(&mysterium.ResidentCountryUpdateRequest{Country: "UK"})
		require.Error(t, err, "identity is required")
	})

	t.Run("Test filter preset", func(t *testing.T) {
		// given
		byName := func(presets []mysterium.ProposalFilterPreset, name string) bool {
			for _, p := range presets {
				if p.Name == name {
					return true
				}
			}
			return false
		}

		// when
		bytes, err := node.ListProposalFilterPresets()
		require.NoError(t, err)

		var presets []mysterium.ProposalFilterPreset
		err = json.Unmarshal(bytes, &presets)
		require.NoError(t, err)

		// when
		for _, name := range []string{"Media Streaming", "Browsing", "Download"} {
			require.True(t, byName(presets, name), "missing name '%s' in preset filters", name)
		}
	})

	t.Run("Test shutdown", func(t *testing.T) {
		err := node.Shutdown()
		require.NoError(t, err)
	})
}

type identityKeystore struct {
	Address string                 `json:"address"`
	Crypto  map[string]interface{} `json:"crypto"`
	ID      string                 `json:"id"`
	Version int                    `json:"version"`
}
