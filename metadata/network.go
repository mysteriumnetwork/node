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
	DiscoveryAPIAddress     string
	BrokerAddress           string
	EtherClientRPC          string
	QualityOracle           string
	ELKAddress              string
	PaymentsContractAddress common.Address
}

// TestnetDefinition defines parameters for test network (currently default network)
var TestnetDefinition = NetworkDefinition{
	"https://testnet-api.mysterium.network/v1",
	"nats://testnet-broker.mysterium.network",
	"https://ropsten.infura.io",
	"https://testnet-morqa.mysterium.network/api/v1",
	"http://metrics.mysterium.network:8091",
	common.HexToAddress("0xbe5F9CCea12Df756bF4a5Baf4c29A10c3ee7C83B"),
}

// LocalnetDefinition defines parameters for local network (expects discovery, broker, morqa and elk services on localhost)
var LocalnetDefinition = NetworkDefinition{
	"http://localhost/v1",
	"localhost",
	"http://localhost:8545",
	"http://localhost:8080",
	"http://localhost:8091",
	common.HexToAddress("0x1955141ba8e77a5B56efBa8522034352c94f77Ea"),
}

// DefaultNetwork defines default network values when no runtime parameters are given
var DefaultNetwork = TestnetDefinition
