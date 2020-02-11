/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package config

import (
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/urfave/cli/v2"
)

var (
	// FlagTestnet uses test network.
	FlagTestnet = cli.BoolFlag{
		Name:  "testnet",
		Usage: "Defines test network configuration",
	}
	// FlagLocalnet uses local network.
	FlagLocalnet = cli.BoolFlag{
		Name:  "localnet",
		Usage: "Defines network configuration which expects locally deployed broker and discovery services",
	}
	// FlagAPIAddress Mysterium API URL
	FlagAPIAddress = cli.StringFlag{
		Name:  "api.address",
		Usage: "URL of Mysterium API",
		Value: metadata.DefaultNetwork.MysteriumAPIAddress,
	}
	// FlagBrokerAddress message broker URI.
	FlagBrokerAddress = cli.StringFlag{
		Name:  "broker-address",
		Usage: "URI of message broker",
		Value: metadata.DefaultNetwork.BrokerAddress,
	}
	// FlagEtherRPC URL or IPC socket to connect to Ethereum node.
	FlagEtherRPC = cli.StringFlag{
		Name:  "ether.client.rpc",
		Usage: "URL or IPC socket to connect to ethereum node, anything what ethereum client accepts - works",
		Value: metadata.DefaultNetwork.EtherClientRPC,
	}
	// FlagNATPunching enables NAT hole punching.
	FlagNATPunching = cli.BoolFlag{
		Name:  "experiment-natpunching",
		Usage: "Enables NAT hole punching",
		Value: true,
	}
)

// RegisterFlagsNetwork function register network flags to flag list
func RegisterFlagsNetwork(flags *[]cli.Flag) {
	*flags = append(
		*flags,
		&FlagTestnet,
		&FlagLocalnet,
		&FlagNATPunching,
		&FlagAPIAddress,
		&FlagBrokerAddress,
		&FlagEtherRPC,
	)
}

// ParseFlagsNetwork function fills in directory options from CLI context
func ParseFlagsNetwork(ctx *cli.Context) {
	Current.ParseBoolFlag(ctx, FlagTestnet)
	Current.ParseBoolFlag(ctx, FlagLocalnet)
	Current.ParseStringFlag(ctx, FlagAPIAddress)
	Current.ParseStringFlag(ctx, FlagBrokerAddress)
	Current.ParseStringFlag(ctx, FlagEtherRPC)
	Current.ParseBoolFlag(ctx, FlagNATPunching)
}
