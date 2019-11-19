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
	"gopkg.in/urfave/cli.v1"
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
	// FlagIdentityCheck enables experimental identity check.
	FlagIdentityCheck = cli.BoolFlag{
		Name:  "experiment-identity-check",
		Usage: "Enables experimental identity check",
	}
	// FlagAPIAddress Mysterium API URL
	FlagAPIAddress = cli.StringFlag{
		Name:  "api.address",
		Usage: "URL of Mysterium API",
		Value: metadata.DefaultNetwork.MysteriumAPIAddress,
	}
	// FlagAccessPolicyAddress Trust oracle URL for retrieving access policies.
	FlagAccessPolicyAddress = cli.StringFlag{
		Name:  "access-policy-address",
		Usage: "URL of trust oracle endpoint for retrieving lists of access policies",
		Value: metadata.DefaultNetwork.AccessPolicyOracleAddress,
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
	// FlagQualityOracleAddress Quality oracle URL.
	FlagQualityOracleAddress = cli.StringFlag{
		Name:  "quality-oracle.address",
		Usage: "Address of the quality oracle service",
		Value: metadata.DefaultNetwork.QualityOracle,
	}
	// FlagNATPunching enables NAT hole punching.
	FlagNATPunching = cli.BoolTFlag{
		Name:  "experiment-natpunching",
		Usage: "Enables NAT hole punching",
	}
)

// RegisterFlagsNetwork function register network flags to flag list
func RegisterFlagsNetwork(flags *[]cli.Flag) {
	*flags = append(
		*flags,
		FlagTestnet,
		FlagLocalnet,
		FlagIdentityCheck,
		FlagNATPunching,
		FlagAPIAddress,
		FlagBrokerAddress,
		FlagEtherRPC,
		FlagQualityOracleAddress,
		FlagAccessPolicyAddress,
	)
}

// ParseFlagsNetwork function fills in directory options from CLI context
func ParseFlagsNetwork(ctx *cli.Context) {
	Current.ParseBoolFlag(ctx, FlagTestnet)
	Current.ParseBoolFlag(ctx, FlagLocalnet)
	Current.ParseBoolFlag(ctx, FlagIdentityCheck)
	Current.ParseStringFlag(ctx, FlagAPIAddress)
	Current.ParseStringFlag(ctx, FlagAccessPolicyAddress)
	Current.ParseStringFlag(ctx, FlagBrokerAddress)
	Current.ParseStringFlag(ctx, FlagEtherRPC)
	Current.ParseStringFlag(ctx, FlagQualityOracleAddress)
	Current.ParseBoolTFlag(ctx, FlagNATPunching)
}
