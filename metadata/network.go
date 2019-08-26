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

import (
	"github.com/ethereum/go-ethereum/common"
)

// NetworkDefinition structure holds all parameters which describe particular network
type NetworkDefinition struct {
	MysteriumAPIAddress       string
	AccessPolicyOracleAddress string
	BrokerAddress             string
	EtherClientRPC            string
	QualityOracle             string
	PaymentsContractAddress   common.Address
	TransactorAddress         string
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
	EtherClientRPC:            "https://ropsten.infura.io/v3/ea3bdc150d434899adb4e178522fb8e1",
	QualityOracle:             "https://testnet-morqa.mysterium.network/api/v1",
	PaymentsContractAddress:   common.HexToAddress("0xE6b3a5c92e7c1f9543A0aEE9A93fE2F6B584c1f7"),
	TransactorAddress:         "https://testnet-transactor.mysterium.network/api/v1",
	RegistryAddress:           "0xE6b3a5c92e7c1f9543A0aEE9A93fE2F6B584c1f7",
	AccountantID:              "0xf28DB7aDf64A2811202B149aa4733A1FB9100e5c",
	ChannelImplAddress:        "0xa26b684d8dBa935DD34544FBd3Ab4d7FDe1C4D07",
	MMNAddress:                "https://my.mysterium.network/api/v1",
}

// LocalnetDefinition defines parameters for local network
// Expects discovery, broker and morqa services on localhost
var LocalnetDefinition = NetworkDefinition{
	MysteriumAPIAddress:       "http://localhost/v1",
	AccessPolicyOracleAddress: "https://localhost:8081/api/v1/access-policies/",
	BrokerAddress:             "localhost",
	EtherClientRPC:            "http://localhost:8545",
	QualityOracle:             "http://localhost:8080",
	PaymentsContractAddress:   common.HexToAddress("0x1955141ba8e77a5B56efBa8522034352c94f77Ea"),
	MMNAddress:                "http://localhost/api/v1",
}

// DefaultNetwork defines default network values when no runtime parameters are given
var DefaultNetwork = TestnetDefinition
