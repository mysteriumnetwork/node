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
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/urfave/cli"
)

var (
	testFlag = cli.BoolFlag{
		Name:  "testnet",
		Usage: "Defines test network configuration",
	}
	localnetFlag = cli.BoolFlag{
		Name:  "localnet",
		Usage: "Defines network configuration which expects locally deployed broker and discovery services",
	}

	identityCheckFlag = cli.BoolFlag{
		Name:  "experiment-identity-check",
		Usage: "Enables experimental identity check",
	}

	paymentCheckFlag = cli.BoolFlag{
		Name:  "experiment-payments",
		Usage: "Enables experimental payments check",
	}

	discoveryAddressFlag = cli.StringFlag{
		Name:  "discovery-address",
		Usage: "`URL` of discovery service",
		Value: metadata.DefaultNetwork.DiscoveryAPIAddress,
	}
	brokerAddressFlag = cli.StringFlag{
		Name:  "broker-address",
		Usage: "`URI` of message broker",
		Value: metadata.DefaultNetwork.BrokerAddress,
	}

	etherRPCFlag = cli.StringFlag{
		Name:  "ether.client.rpc",
		Usage: "URL or IPC socket to connect to ethereum node, anything what ethereum client accepts - works",
		Value: metadata.DefaultNetwork.EtherClientRPC,
	}
	etherContractPaymentsFlag = cli.StringFlag{
		Name:  "ether.contract.payments",
		Usage: "Address of payments contract",
		Value: metadata.DefaultNetwork.PaymentsContractAddress.String(),
	}

	qualityOracleFlag = cli.StringFlag{
		Name:  "quality-oracle.address",
		Usage: "Address of the quality oracle service",
		Value: metadata.DefaultNetwork.QualityOracle,
	}

	metricsDisableFlag = cli.BoolFlag{
		Name:  "metrics.disable",
		Usage: "Opt-out from sending usage metrics",
	}
	metricsAddressFlag = cli.StringFlag{
		Name:  "metrics.address",
		Usage: "Address of metrics service",
		Value: metadata.DefaultNetwork.MetricsAddress,
	}
)

// RegisterFlagsNetwork function register network flags to flag list
func RegisterFlagsNetwork(flags *[]cli.Flag) {
	*flags = append(
		*flags,
		testFlag, localnetFlag,
		identityCheckFlag,
		paymentCheckFlag,
		discoveryAddressFlag, brokerAddressFlag,
		etherRPCFlag, etherContractPaymentsFlag,
		qualityOracleFlag,
		metricsAddressFlag,
		metricsDisableFlag,
	)
}

// ParseFlagsNetwork function fills in directory options from CLI context
func ParseFlagsNetwork(ctx *cli.Context) node.OptionsNetwork {
	return node.OptionsNetwork{
		ctx.GlobalBool(testFlag.Name),
		ctx.GlobalBool(localnetFlag.Name),

		ctx.GlobalBool(identityCheckFlag.Name),
		ctx.GlobalBool(paymentCheckFlag.Name),

		ctx.GlobalString(discoveryAddressFlag.Name),
		ctx.GlobalString(brokerAddressFlag.Name),

		ctx.GlobalString(etherRPCFlag.Name),
		ctx.GlobalString(etherContractPaymentsFlag.Name),

		ctx.GlobalString(qualityOracleFlag.Name),
		ctx.GlobalBool(metricsDisableFlag.Name),
		ctx.GlobalString(metricsAddressFlag.Name),
	}
}
