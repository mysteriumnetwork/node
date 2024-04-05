/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package metadata

import "fmt"

// NetworkDefinition structure holds all parameters which describe particular network
type NetworkDefinition struct {
	// Deprecated: use DiscoveryAddress
	MysteriumAPIAddress       string
	DiscoveryAddress          string
	AccessPolicyOracleAddress string
	BrokerAddresses           []string
	TransactorAddress         string
	AffiliatorAddress         string
	Chain1                    ChainDefinition
	Chain2                    ChainDefinition
	MMNAddress                string
	MMNAPIAddress             string
	PilvytisAddress           string
	ObserverAddress           string
	DNSMap                    map[string][]string
	DefaultChainID            int64
	DefaultCurrency           string
	LocationAddress           string
	Payments                  Payments
}

// ChainDefinition defines the configuration for the chain.
type ChainDefinition struct {
	RegistryAddress    string
	HermesID           string
	ChannelImplAddress string
	ChainID            int64
	MystAddress        string
	EtherClientRPC     []string
	KnownHermeses      []string
}

// Payments defines payments configuration
type Payments struct {
	DataLeewayMegabytes uint64
}

// MainnetDefinition defines parameters for mainnet network (currently default network)
var MainnetDefinition = NetworkDefinition{
	MysteriumAPIAddress:       "https://discovery.mysterium.network/api/v4",
	DiscoveryAddress:          "https://discovery.mysterium.network/api/v4",
	AccessPolicyOracleAddress: "https://trust.mysterium.network/api/v1/access-policies/",
	BrokerAddresses:           []string{"nats://broker.mysterium.network"},
	TransactorAddress:         "https://transactor.mysterium.network/api/v1",
	AffiliatorAddress:         "https://affiliator.mysterium.network/api/v1",
	Chain1: ChainDefinition{ // TODO: Update when mainnet deployed.
		RegistryAddress:    "0x87F0F4b7e0FAb14A565C87BAbbA6c40c92281b51",
		ChannelImplAddress: "0xbd20839b331a7a8d10e34cdf7219edf334814c4f",
		HermesID:           "0xa62a2a75949d25e17c6f08a7818e7be97c18a8d2",
		ChainID:            1,
		MystAddress:        "0x4Cf89ca06ad997bC732Dc876ed2A7F26a9E7f361",
		EtherClientRPC: []string{
			"https://ethereum1.mysterium.network/",
			"https://cloudflare-eth.com/",
		},
		KnownHermeses: []string{
			"0xa62a2a75949d25e17c6f08a7818e7be97c18a8d2",
		},
	},
	Chain2: ChainDefinition{
		RegistryAddress:    "0x87F0F4b7e0FAb14A565C87BAbbA6c40c92281b51",
		ChannelImplAddress: "0x6b423D3885B4877b5760E149364f85f185f477aD",
		HermesID:           "0x80ed28d84792d8b153bf2f25f0c4b7a1381de4ab",
		ChainID:            137,
		MystAddress:        "0x1379e8886a944d2d9d440b3d88df536aea08d9f3",
		EtherClientRPC: []string{
			"https://polygon1.mysterium.network/",
			"https://polygon-rpc.com/",
		},
		KnownHermeses: []string{
			"0xa62a2a75949d25e17c6f08a7818e7be97c18a8d2",
			"0x80ed28d84792d8b153bf2f25f0c4b7a1381de4ab",
		},
	},
	MMNAddress:      "https://my.mystnodes.com",
	MMNAPIAddress:   "https://my.mystnodes.com/api/v1",
	PilvytisAddress: "https://pilvytis.mysterium.network",
	ObserverAddress: "https://observer.mysterium.network",
	DNSMap: map[string][]string{
		"discovery.mysterium.network":  {"51.158.204.30", "51.158.204.75", "51.158.204.9", "51.158.204.23"},
		"trust.mysterium.network":      {"51.158.204.30", "51.158.204.75", "51.158.204.9", "51.158.204.23"},
		"broker.mysterium.network":     {"51.158.204.30", "51.158.204.75", "51.158.204.9", "51.158.204.23"},
		"transactor.mysterium.network": {"51.158.204.30", "51.158.204.75", "51.158.204.9", "51.158.204.23"},
		"affiliator.mysterium.network": {"51.158.204.30", "51.158.204.75", "51.158.204.9", "51.158.204.23"},
		"pilvytis.mysterium.network":   {"51.158.204.30", "51.158.204.75", "51.158.204.9", "51.158.204.23"},
		"observer.mysterium.network":   {"51.158.204.30", "51.158.204.75", "51.158.204.9", "51.158.204.23"},
	},
	DefaultChainID:  137,
	DefaultCurrency: "MYST",
	LocationAddress: "https://location.mysterium.network/api/v1/location",
	Payments: Payments{
		DataLeewayMegabytes: 20,
	},
}

// LocalnetDefinition defines parameters for local network
// Expects discovery, broker and morqa services on localhost
var LocalnetDefinition = NetworkDefinition{
	MysteriumAPIAddress:       "http://localhost:8001/v1",
	DiscoveryAddress:          "http://localhost:8001/v1",
	AccessPolicyOracleAddress: "https://localhost:8081/api/v1/access-policies/",
	BrokerAddresses:           []string{"localhost"},
	MMNAddress:                "http://localhost/",
	MMNAPIAddress:             "http://localhost/api/v1",
	PilvytisAddress:           "http://localhost:8002/api/v1",
	DNSMap: map[string][]string{
		"localhost": {"127.0.0.1"},
	},
	DefaultChainID: 1,
}

// TestnetDefinition defines parameters for testnet network
var TestnetDefinition = NetworkDefinition{
	MysteriumAPIAddress:       "https://discovery-testnet.mysterium.network/api/v4",
	DiscoveryAddress:          "https://discovery-testnet.mysterium.network/api/v4",
	AccessPolicyOracleAddress: "https://trust-testnet.mysterium.network/api/v1/access-policies/",
	BrokerAddresses:           []string{"nats://broker.mysterium.network:4223"},
	TransactorAddress:         "https://transactor-testnet.mysterium.network/api/v1",
	AffiliatorAddress:         "https://affiliator.mysterium.network/api/v1",
	Chain1: ChainDefinition{
		ChainID: 1,
		EtherClientRPC: []string{
			"https://ethereum1.mysterium.network/",
			"https://cloudflare-eth.com/",
		},
	},
	Chain2: ChainDefinition{
		RegistryAddress:    "0x1ba2DF26371E83D87Afee2F27a42f5A7FE9e5219",
		ChannelImplAddress: "0x6FE3E5e5008e49821BF7282870eC831BA9694dDB",
		HermesID:           "0xcAeF9A6E9C2d9C0Ee3333529922c280580365b51",
		ChainID:            80001,
		MystAddress:        "0xB923b52b60E247E34f9afE6B3fa5aCcBAea829E8",
		EtherClientRPC: []string{
			"https://polygon-mumbai1.mysterium.network",
		},
		KnownHermeses: []string{
			"0xcAeF9A6E9C2d9C0Ee3333529922c280580365b51",
		},
	},
	MMNAddress:      "https://my.mystnodes.com",
	MMNAPIAddress:   "https://my.mystnodes.com/api/v1",
	PilvytisAddress: "https://pilvytis-testnet.mysterium.network",
	ObserverAddress: "https://observer-testnet.mysterium.network",
	DNSMap: map[string][]string{
		"trust.mysterium.network":      {"51.158.204.30", "51.158.204.75", "51.158.204.9", "51.158.204.23"},
		"broker.mysterium.network":     {"51.158.204.30", "51.158.204.75", "51.158.204.9", "51.158.204.23"},
		"affiliator.mysterium.network": {"51.158.204.30", "51.158.204.75", "51.158.204.9", "51.158.204.23"},
	},
	DefaultChainID:  80001,
	DefaultCurrency: "MYST",
	LocationAddress: "https://location.mysterium.network/api/v1/location",
	Payments: Payments{
		DataLeewayMegabytes: 20,
	},
}

// GetDefaultFlagValues returns a map of flag name to default value for the network
func (n *NetworkDefinition) GetDefaultFlagValues() map[string]any {
	res := map[string]any{
		FlagNames.MysteriumAPIAddress:       n.MysteriumAPIAddress,
		FlagNames.DiscoveryAddress:          n.DiscoveryAddress,
		FlagNames.AccessPolicyOracleAddress: n.AccessPolicyOracleAddress,
		FlagNames.BrokerAddressesFlag:       n.BrokerAddresses,
		FlagNames.TransactorAddress:         n.TransactorAddress,
		FlagNames.AffiliatorAddress:         n.AffiliatorAddress,

		FlagNames.MMNAddress:                  n.MMNAddress,
		FlagNames.MMNAPIAddress:               n.MMNAPIAddress,
		FlagNames.PilvytisAddress:             n.PilvytisAddress,
		FlagNames.ObserverAddress:             n.ObserverAddress,
		FlagNames.DefaultChainIDFlag:          n.DefaultChainID,
		FlagNames.DefaultCurrency:             n.DefaultCurrency,
		FlagNames.LocationAddress:             n.LocationAddress,
		FlagNames.PaymentsDataLeewayMegabytes: n.Payments.DataLeewayMegabytes,
	}

	//chain 1
	for k, v := range n.getChainFlagValues(1) {
		res[k] = v
	}
	//chain 2
	for k, v := range n.getChainFlagValues(2) {
		res[k] = v
	}

	return res
}

// NetworkDefinitionFlagNames structure holds all parameters which define the flags for a network
type NetworkDefinitionFlagNames struct {
	NetworkDefinition
	BrokerAddressesFlag         string
	Chain1Flag                  ChainDefinitionFlagNames
	Chain2Flag                  ChainDefinitionFlagNames
	DefaultChainIDFlag          string
	PaymentsDataLeewayMegabytes string
}

// ChainDefinitionFlagNames structure holds all parameters which define the flags for a chain
type ChainDefinitionFlagNames struct {
	ChainDefinition
	ChainIDFlag        string
	EtherClientRPCFlag string
	KnownHermesesFlag  string
}

// FlagNames defines the flag that sets each network parameter
var FlagNames = NetworkDefinitionFlagNames{
	NetworkDefinition: NetworkDefinition{
		MysteriumAPIAddress:       "api.address",
		DiscoveryAddress:          "discovery.address",
		AccessPolicyOracleAddress: "access-policy.address",
		TransactorAddress:         "transactor.address",
		AffiliatorAddress:         "affiliator.address",
		MMNAddress:                "mmn.web-address",
		MMNAPIAddress:             "mmn.api-address",
		PilvytisAddress:           "pilvytis.address",
		ObserverAddress:           "observer.address",
		DefaultCurrency:           "default-currency",
		LocationAddress:           "location.address",
	},
	BrokerAddressesFlag:         "broker-address",
	Chain1Flag:                  getChainDefinitionFlagNames(1),
	Chain2Flag:                  getChainDefinitionFlagNames(2),
	DefaultChainIDFlag:          "chain-id",
	PaymentsDataLeewayMegabytes: "payments.consumer.data-leeway-megabytes",
}

// DefaultNetwork defines default network values when no runtime parameters are given
var DefaultNetwork = MainnetDefinition

func getChainDefinitionFlagNames(chainIndex int) ChainDefinitionFlagNames {
	return ChainDefinitionFlagNames{
		ChainDefinition: ChainDefinition{
			RegistryAddress:    fmt.Sprintf("chains.%v.registry", chainIndex),
			HermesID:           fmt.Sprintf("chains.%v.hermes", chainIndex),
			ChannelImplAddress: fmt.Sprintf("chains.%v.channelImplementation", chainIndex),
			MystAddress:        fmt.Sprintf("chains.%v.myst", chainIndex),
		},
		ChainIDFlag:        fmt.Sprintf("chains.%v.chainID", chainIndex),
		EtherClientRPCFlag: fmt.Sprintf("ether.client.rpcl%v", chainIndex),
		KnownHermesesFlag:  fmt.Sprintf("chains.%v.knownHermeses", chainIndex),
	}
}

func (n *NetworkDefinition) getChainFlagValues(chainIndex int) map[string]any {
	chainDefinition := n.Chain1
	if chainIndex == 2 {
		chainDefinition = n.Chain2
	}

	flagNames := FlagNames.Chain1Flag
	if chainIndex == 2 {
		flagNames = FlagNames.Chain2Flag
	}

	return map[string]any{
		flagNames.RegistryAddress:    chainDefinition.RegistryAddress,
		flagNames.HermesID:           chainDefinition.HermesID,
		flagNames.ChannelImplAddress: chainDefinition.ChannelImplAddress,
		flagNames.ChainIDFlag:        chainDefinition.ChainID,
		flagNames.MystAddress:        chainDefinition.MystAddress,
		flagNames.EtherClientRPCFlag: chainDefinition.EtherClientRPC,
		flagNames.KnownHermesesFlag:  chainDefinition.KnownHermeses,
	}
}
