/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package node

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/metadata"
)

// NetworkOptions describes possible parameters of network configuration
type NetworkOptions struct {
	Testnet  bool
	Localnet bool

	DiscoveryAPIAddress string
	BrokerAddress       string

	EtherClientRPC       string
	EtherPaymentsAddress string
}

// GetNetworkDefinition function returns network definition combined from testnet/localnet flags and possible overrides
func GetNetworkDefinition(options NetworkOptions) metadata.NetworkDefinition {
	network := metadata.DefaultNetwork

	switch {
	case options.Testnet:
		network = metadata.TestnetDefinition
	case options.Localnet:
		network = metadata.LocalnetDefinition
	}

	//override defined values one by one from options
	if options.DiscoveryAPIAddress != metadata.DefaultNetwork.DiscoveryAPIAddress {
		network.DiscoveryAPIAddress = options.DiscoveryAPIAddress
	}

	if options.BrokerAddress != metadata.DefaultNetwork.BrokerAddress {
		network.BrokerAddress = options.BrokerAddress
	}

	normalizedAddress := common.HexToAddress(options.EtherPaymentsAddress)
	if normalizedAddress.String() != metadata.DefaultNetwork.PaymentsContractAddress.String() {
		network.PaymentsContractAddress = common.HexToAddress(options.EtherPaymentsAddress)
	}

	if options.EtherClientRPC != metadata.DefaultNetwork.EtherClientRPC {
		network.EtherClientRPC = options.EtherClientRPC
	}

	return network
}
