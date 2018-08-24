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

package cmd

import (
	"flag"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysterium/node/metadata"
)

// NetworkOptions describes possible parameters of network configuration
type NetworkOptions struct {
	DiscoveryAPIAddress  string
	BrokerAddress        string
	Localnet             bool
	Testnet              bool
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

// ParseNetworkOptions function parses (or registers) network options from flag library
func ParseNetworkOptions(flags *flag.FlagSet, options *NetworkOptions) {
	flags.StringVar(
		&options.DiscoveryAPIAddress,
		"discovery-address",
		metadata.DefaultNetwork.DiscoveryAPIAddress,
		"Address (URL form) of discovery service",
	)

	flags.StringVar(
		&options.BrokerAddress,
		"broker-address",
		metadata.DefaultNetwork.BrokerAddress,
		"Address (IP or domain name) of message broker",
	)

	flags.BoolVar(
		&options.Localnet,
		"localnet",
		false,
		"Defines network configuration which expects localy deployed broker and discovery services",
	)

	flags.BoolVar(
		&options.Testnet,
		"testnet",
		false,
		"Defines test network configuration",
	)

	flags.StringVar(
		&options.EtherClientRPC,
		"ether.client.rpc",
		metadata.DefaultNetwork.EtherClientRPC,
		"Url or IPC socket to connect to ethereum node, anything what ethereum client accepts - works",
	)

	flags.StringVar(
		&options.EtherPaymentsAddress,
		"ether.contract.payments",
		metadata.DefaultNetwork.PaymentsContractAddress.String(),
		"Address of payments contract",
	)
}
