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
	BrokerAddress             string
	EtherClientRPC            string
	QualityOracle             string
	TransactorAddress         string
	AccountantAddress         string
	RegistryAddress           string
	AccountantID              string
	ChannelImplAddress        string
	MMNAddress                string
}

// TestnetDefinition defines parameters for test network (currently default network)
var TestnetDefinition = NetworkDefinition{
	MysteriumAPIAddress:       "https://testnet-api.mysterium.network/v1",
	AccessPolicyOracleAddress: "https://testnet-trust.mysterium.network/api/v1/access-policies/",
	BrokerAddress:             "nats://testnet-broker.mysterium.network",
	EtherClientRPC:            "wss://goerli.infura.io/ws/v3/1afe7cfb9b4b4610b3905d77ee2ed64d",
	QualityOracle:             "https://testnet-morqa.mysterium.network/api/v1",
	TransactorAddress:         "https://testnet-transactor.mysterium.network/api/v1",
	RegistryAddress:           "0x611ad702f6A55C16A1bA6733a20D457488B5EAaF",
	ChannelImplAddress:        "0x5488774D8c7D170D4a8ecA89892c54b8DEca510b",
	AccountantID:              "0x0324814fdeFdffD8BE5774a5EaDD70F35A1F1775",
	AccountantAddress:         "https://testnet-accountant.mysterium.network/api/v1",
	MMNAddress:                "https://my.mysterium.network/api/v1",
}

// LocalnetDefinition defines parameters for local network
// Expects discovery, broker and morqa services on localhost
var LocalnetDefinition = NetworkDefinition{
	MysteriumAPIAddress:       "http://localhost:8001/v1",
	AccessPolicyOracleAddress: "https://localhost:8081/api/v1/access-policies/",
	BrokerAddress:             "localhost",
	EtherClientRPC:            "http://localhost:8545",
	QualityOracle:             "http://localhost:8085",
	MMNAddress:                "http://localhost/api/v1",
}

// DefaultNetwork defines default network values when no runtime parameters are given
var DefaultNetwork = TestnetDefinition
