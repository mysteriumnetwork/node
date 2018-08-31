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

	"github.com/mysterium/node/core/node"
	"github.com/mysterium/node/metadata"
	"github.com/urfave/cli"
)

// ParseNetworkArguments function parses (or registers) network options from flag library
func ParseNetworkArguments(flags *flag.FlagSet, options *node.NetworkOptions) {
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
		"Defines network configuration which expects locally deployed broker and discovery services",
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

var (
	testFlag = cli.BoolFlag{
		Name:  "testnet",
		Usage: "Defines test network configuration",
	}
	localnetFlag = cli.BoolFlag{
		Name:  "localnet",
		Usage: "Defines network configuration which expects locally deployed broker and discovery services",
	}

	discoveryAddressFlag = cli.StringFlag{
		Name:  "discovery-address",
		Usage: "Address (URL form) of discovery service",
		Value: metadata.DefaultNetwork.DiscoveryAPIAddress,
	}
	brokerAddressFlag = cli.StringFlag{
		Name:  "broker-address",
		Usage: "Address (IP or domain name) of message broker",
		Value: metadata.DefaultNetwork.BrokerAddress,
	}

	etherRpcFlag = cli.StringFlag{
		Name:  "ether.client.rpc",
		Usage: "Url or IPC socket to connect to ethereum node, anything what ethereum client accepts - works",
		Value: metadata.DefaultNetwork.EtherClientRPC,
	}
	etherContractPaymentsFlag = cli.StringFlag{
		Name:  "ether.contract.payments",
		Usage: "Address of payments contract",
		Value: metadata.DefaultNetwork.PaymentsContractAddress.String(),
	}
)

// RegisterNetworkFlags function register network flags to flag list
func RegisterNetworkFlags(flags *[]cli.Flag) {
	*flags = append(
		*flags,
		testFlag, localnetFlag,
		discoveryAddressFlag, brokerAddressFlag,
		etherRpcFlag, etherContractPaymentsFlag,
	)
}

// ParseNetworkFlags function fills in directory options from CLI context
func ParseNetworkFlags(ctx *cli.Context) node.NetworkOptions {
	return node.NetworkOptions{
		ctx.GlobalBool(testFlag.Name),
		ctx.GlobalBool(localnetFlag.Name),

		ctx.GlobalString(discoveryAddressFlag.Name),
		ctx.GlobalString(brokerAddressFlag.Name),

		ctx.GlobalString(etherRpcFlag.Name),
		ctx.GlobalString(etherContractPaymentsFlag.Name),
	}
}
