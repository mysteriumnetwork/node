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
}

// TestnetDefinition defines parameters for test network (currently default network)
var TestnetDefinition = NetworkDefinition{
	MysteriumAPIAddress:       "https://testnet-api.mysterium.network/v1",
	AccessPolicyOracleAddress: "https://testnet-trust.mysterium.network/api/v1/access-policies/",
	BrokerAddress:             "nats://testnet-broker.mysterium.network",
	EtherClientRPC:            "https://ropsten.infura.io",
	QualityOracle:             "https://testnet-morqa.mysterium.network/api/v1",
	PaymentsContractAddress:   common.HexToAddress("0xbe5F9CCea12Df756bF4a5Baf4c29A10c3ee7C83B"),
	TransactorAddress:         "https://testnet-transactor.mysterium.network/api/v1",
	RegistryAddress:           "0xE6b3a5c92e7c1f9543A0aEE9A93fE2F6B584c1f7",
	AccountantID:              "0x5a10863FeB6f3BcaEe9C960D0c24F4887cF15053",
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
}

// DefaultNetwork defines default network values when no runtime parameters are given
var DefaultNetwork = TestnetDefinition
