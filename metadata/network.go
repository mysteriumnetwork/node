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

// NetworkDefinition structure holds all parameters which describe particular network
type NetworkDefinition struct {
	MysteriumAPIAddress       string
	AccessPolicyOracleAddress string
	BrokerAddresses           []string
	TransactorAddress         string
	AffiliatorAddress         string
	TransactorIdentity        string
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
	Testnet3HermesURL         string
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
	Consumer Consumer
}

// Consumer defines consumer side settings
type Consumer struct {
	DataLeewayMegabytes uint64
	PriceGiBMax         string
	PriceHourMax        string
	EtherClientRPC      string
}

// MainnetDefinition defines parameters for mainnet network (currently default network)
var MainnetDefinition = NetworkDefinition{
	MysteriumAPIAddress:       "https://discovery.mysterium.network/api/v3",
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
		ChannelImplAddress: "0x813d3A0ef42FD4F25F2854811A64D5842EF3F8D1",
		HermesID:           "0xDe82990405aCc36B4Fd53c94A24D1010fcc1F83d",
		ChainID:            137,
		MystAddress:        "0x1379e8886a944d2d9d440b3d88df536aea08d9f3",
		EtherClientRPC: []string{
			"https://polygon1.mysterium.network/",
			"https://polygon-rpc.com/",
		},
		KnownHermeses: []string{
			"0xa62a2a75949d25e17c6f08a7818e7be97c18a8d2",
			"0xde82990405acc36b4fd53c94a24d1010fcc1f83d",
		},
	},
	MMNAddress:      "https://mystnodes.com",
	MMNAPIAddress:   "https://mystnodes.com/api/v1",
	PilvytisAddress: "https://pilvytis.mysterium.network",
	ObserverAddress: "https://observer.mysterium.network",
	DNSMap: map[string][]string{
		"discovery.mysterium.network":  {"51.15.116.186", "51.15.72.87"},
		"trust.mysterium.network":      {"51.15.116.186", "51.15.72.87"},
		"broker.mysterium.network":     {"51.15.116.186", "51.15.72.87"},
		"transactor.mysterium.network": {"51.15.116.186", "51.15.72.87"},
		"affiliator.mysterium.network": {"51.15.116.186", "51.15.72.87"},
		"pilvytis.mysterium.network":   {"51.15.116.186", "51.15.72.87"},
		"observer.mysterium.network":   {"51.15.116.186", "51.15.72.87"},
	},
	DefaultChainID:    137,
	DefaultCurrency:   "MYST",
	LocationAddress:   "https://location.mysterium.network/api/v1/location",
	Testnet3HermesURL: "https://testnet3-hermes.mysterium.network/api/v1",
	Payments: Payments{
		Consumer: Consumer{
			DataLeewayMegabytes: 20,
			PriceGiBMax:         "500000000000000000", // 0.5 MYSTT
			PriceHourMax:        "180000000000000",    // 0.0018 MYSTT
		},
	},
}

// LocalnetDefinition defines parameters for local network
// Expects discovery, broker and morqa services on localhost
var LocalnetDefinition = NetworkDefinition{
	MysteriumAPIAddress:       "http://localhost:8001/v1",
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

// DefaultNetwork defines default network values when no runtime parameters are given
var DefaultNetwork = MainnetDefinition
