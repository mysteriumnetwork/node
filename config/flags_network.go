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
	"github.com/urfave/cli/v2"

	"github.com/mysteriumnetwork/node/metadata"
)

var (
	// FlagLocalnet uses local network.
	FlagLocalnet = cli.BoolFlag{
		Name:  "localnet",
		Usage: "Defines network configuration which expects locally deployed broker and discovery services",
	}
	// FlagTestnet2 uses testnet2 network.
	FlagTestnet2 = cli.BoolFlag{
		Name:  "testnet2",
		Usage: "Defines testnet2 configuration",
		Value: true,
	}
	// FlagAPIAddress Mysterium API URL
	FlagAPIAddress = cli.StringFlag{
		Name:  "api.address",
		Usage: "URL of Mysterium API",
		Value: metadata.DefaultNetwork.MysteriumAPIAddress,
	}
	// FlagChainID chain id to use
	FlagChainID = cli.Int64Flag{
		Name:  "chain-id",
		Usage: "The chain ID to use",
		Value: metadata.DefaultNetwork.DefaultChainID,
	}
	// FlagBrokerAddress message broker URI.
	FlagBrokerAddress = cli.StringSliceFlag{
		Name:  "broker-address",
		Usage: "URI of message broker",
		Value: cli.NewStringSlice(metadata.DefaultNetwork.BrokerAddresses...),
	}
	// FlagEtherRPCL1 URL or IPC socket to connect to Ethereum node.
	FlagEtherRPCL1 = cli.StringFlag{
		Name:  "ether.client.rpcl1",
		Usage: "L1 URL or IPC socket to connect to ethereum node, anything what ethereum client accepts - works",
		Value: metadata.DefaultNetwork.Chain1.EtherClientRPC,
	}
	// FlagEtherRPCL2 URL or IPC socket to connect to Ethereum node.
	FlagEtherRPCL2 = cli.StringFlag{
		Name:  "ether.client.rpcl2",
		Usage: "L2 URL or IPC socket to connect to ethereum node, anything what ethereum client accepts - works",
		Value: metadata.DefaultNetwork.Chain2.EtherClientRPC,
	}
	// FlagNATPunching enables NAT hole punching.
	FlagNATPunching = cli.BoolFlag{
		Name:  "experiment-natpunching",
		Usage: "Enables NAT hole punching",
		Value: true,
	}
	// FlagPortMapping enables NAT port mapping.
	FlagPortMapping = cli.BoolFlag{
		Name:  "nat-port-mapping",
		Usage: "Enables NAT port mapping",
		Value: true,
	}
	// FlagIncomingFirewall enables incoming traffic filtering.
	FlagIncomingFirewall = cli.BoolFlag{
		Name:  "incoming-firewall",
		Usage: "Enables incoming traffic filtering",
		Value: false,
	}
	// FlagOutgoingFirewall enables outgoing traffic filtering.
	FlagOutgoingFirewall = cli.BoolFlag{
		Name:  "outgoing-firewall",
		Usage: "Enables outgoing traffic filtering",
		Value: false,
	}
	// FlagKeepConnectedOnFail keeps connection active to prevent traffic leaks.
	FlagKeepConnectedOnFail = cli.BoolFlag{
		Name:  "keep-connected-on-fail",
		Usage: "Do not disconnect consumer on session fail to prevent traffic leaks",
		Value: false,
	}
)

// RegisterFlagsNetwork function register network flags to flag list
func RegisterFlagsNetwork(flags *[]cli.Flag) {
	*flags = append(
		*flags,
		&FlagLocalnet,
		&FlagPortMapping,
		&FlagNATPunching,
		&FlagAPIAddress,
		&FlagBrokerAddress,
		&FlagEtherRPCL1,
		&FlagEtherRPCL2,
		&FlagIncomingFirewall,
		&FlagOutgoingFirewall,
		&FlagTestnet2,
		&FlagChainID,
		&FlagKeepConnectedOnFail,
	)
}

// ParseFlagsNetwork function fills in directory options from CLI context
func ParseFlagsNetwork(ctx *cli.Context) {
	Current.ParseBoolFlag(ctx, FlagLocalnet)
	Current.ParseBoolFlag(ctx, FlagTestnet2)
	Current.ParseStringFlag(ctx, FlagAPIAddress)
	Current.ParseStringSliceFlag(ctx, FlagBrokerAddress)
	Current.ParseStringFlag(ctx, FlagEtherRPCL1)
	Current.ParseStringFlag(ctx, FlagEtherRPCL2)
	Current.ParseBoolFlag(ctx, FlagPortMapping)
	Current.ParseBoolFlag(ctx, FlagNATPunching)
	Current.ParseBoolFlag(ctx, FlagIncomingFirewall)
	Current.ParseBoolFlag(ctx, FlagOutgoingFirewall)
	Current.ParseInt64Flag(ctx, FlagChainID)
	Current.ParseBoolFlag(ctx, FlagKeepConnectedOnFail)
}
