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
	EtherClientRPC            string
	TransactorAddress         string
	TransactorIdentity        string
	Chain1                    ChainDefinition
	Chain2                    ChainDefinition
	MMNAddress                string
	MMNAPIAddress             string
	PilvytisAddress           string
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
}

// Payments defines payments configuration
type Payments struct {
	Consumer Consumer
}

// Consumer defines consumer side settings
type Consumer struct {
	DataLeewayMegabytes uint64
	PricePerGIBMax      string
	PricePerGIBMin      string
	PricePerMinuteMax   string
	PricePerMinuteMin   string
}

// Testnet2Definition defines parameters for testnet2 network (currently default network)
var Testnet2Definition = NetworkDefinition{
	MysteriumAPIAddress:       "https://testnet2-api.mysterium.network/v1",
	AccessPolicyOracleAddress: "https://testnet2-trust.mysterium.network/api/v1/access-policies/",
	BrokerAddresses:           []string{"nats://testnet2-broker.mysterium.network"},
	EtherClientRPC:            "wss://goerli.infura.io/ws/v3/c2c7da73fcc84ec5885a7bb0eb3c3637",
	TransactorIdentity:        "0x45b224f0cd64ed5179502da42ed4e32228485b3b",
	TransactorAddress:         "https://testnet2-transactor.mysterium.network/api/v1",
	Chain1: ChainDefinition{
		RegistryAddress:    "0x15B1281F4e58215b2c3243d864BdF8b9ddDc0DA2",
		ChannelImplAddress: "0xc49B987fB8701a41ae65Cf934a811FeA15bCC6E4",
		HermesID:           "0xD5d2f5729D4581dfacEBedF46C7014DeFda43585",
		ChainID:            5,
		MystAddress:        "0xf74a5ca65E4552CfF0f13b116113cCb493c580C5",
	},
	Chain2: ChainDefinition{
		RegistryAddress:    "0x15B1281F4e58215b2c3243d864BdF8b9ddDc0DA2",
		ChannelImplAddress: "0xc49B987fB8701a41ae65Cf934a811FeA15bCC6E4",
		HermesID:           "0xD5d2f5729D4581dfacEBedF46C7014DeFda43585",
		ChainID:            80001,
		MystAddress:        "0xf74a5ca65E4552CfF0f13b116113cCb493c580C5",
	},
	MMNAddress:      "https://my.mysterium.network/",
	MMNAPIAddress:   "https://my.mysterium.network/api/v1",
	PilvytisAddress: "https://testnet2-pilvytis.mysterium.network/api/v1",
	DNSMap: map[string][]string{
		"testnet2-api.mysterium.network":        {"78.47.55.197"},
		"testnet2-trust.mysterium.network":      {"95.216.204.232"},
		"testnet2-broker.mysterium.network":     {"95.216.204.232"},
		"testnet2-transactor.mysterium.network": {"135.181.82.67"},
		"testnet2-pilvytis.mysterium.network":   {"195.201.220.36"},
		"my.mysterium.network":                  {"138.201.174.94"},
	},
	DefaultChainID:  5,
	DefaultCurrency: "MYSTT",
	LocationAddress: "https://testnet2-location.mysterium.network/api/v1/location",
	Payments: Payments{
		Consumer: Consumer{
			DataLeewayMegabytes: 20,
			PricePerGIBMax:      "500000000000000000", // 0.5 MYSTT
			PricePerGIBMin:      "0",
			PricePerMinuteMax:   "30000000000000", // 0.00003 MYSTT
			PricePerMinuteMin:   "0",
		},
	},
}

// LocalnetDefinition defines parameters for local network
// Expects discovery, broker and morqa services on localhost
var LocalnetDefinition = NetworkDefinition{
	MysteriumAPIAddress:       "http://localhost:8001/v1",
	AccessPolicyOracleAddress: "https://localhost:8081/api/v1/access-policies/",
	BrokerAddresses:           []string{"localhost"},
	EtherClientRPC:            "http://localhost:8545",
	MMNAddress:                "http://localhost/",
	MMNAPIAddress:             "http://localhost/api/v1",
	PilvytisAddress:           "http://localhost:8002/api/v1",
	DNSMap: map[string][]string{
		"localhost": {"127.0.0.1"},
	},
	DefaultChainID: 1,
}

// DefaultNetwork defines default network values when no runtime parameters are given
var DefaultNetwork = Testnet2Definition
