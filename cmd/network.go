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

// ParseNetworkOptions function parses (or registers) network options from flag library
func ParseNetworkOptions(flags *flag.FlagSet, options *node.NetworkOptions) {
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

// RegisterNetworkFlags function register directory options to flag set
func RegisterNetworkFlags(flags *[]cli.Flag, options *node.NodeOptions) error {
	*flags = append(*flags,
		cli.BoolFlag{
			Name:        "testnet",
			Usage:       "Defines test network configuration",
			Destination: &options.Testnet,
		},
		cli.BoolFlag{
			Name:        "localnet",
			Usage:       "Defines network configuration which expects localy deployed broker and discovery services",
			Destination: &options.Localnet,
		},

		cli.StringFlag{
			Name:        "discovery-address",
			Usage:       "Address (URL form) of discovery service",
			Destination: &options.NetworkOptions.DiscoveryAPIAddress,
			Value:       metadata.DefaultNetwork.DiscoveryAPIAddress,
		},
		cli.StringFlag{
			Name:        "broker-address",
			Usage:       "Address (IP or domain name) of message broker",
			Destination: &options.BrokerAddress,
			Value:       metadata.DefaultNetwork.BrokerAddress,
		},

		cli.StringFlag{
			Name:        "ether.client.rpc",
			Usage:       "Url or IPC socket to connect to ethereum node, anything what ethereum client accepts - works",
			Destination: &options.EtherClientRPC,
			Value:       metadata.DefaultNetwork.EtherClientRPC,
		},
		cli.StringFlag{
			Name:        "ether.contract.payments",
			Usage:       "Address of payments contract",
			Destination: &options.EtherPaymentsAddress,
			Value:       metadata.DefaultNetwork.PaymentsContractAddress.String(),
		},
	)
	return nil
}
