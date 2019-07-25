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
	"fmt"

	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/metadata"
	"gopkg.in/urfave/cli.v1"
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

	apiAddressFlag = cli.StringFlag{
		Name:  "api.address",
		Usage: "URL of Mysterium API",
		Value: metadata.DefaultNetwork.MysteriumAPIAddress,
	}
	apiAddressFlagDepreciated = cli.StringFlag{
		Name:  "discovery-address",
		Usage: fmt.Sprintf("URL of Mysterium API (DEPRECIATED, start using '--%s')", apiAddressFlag.Name),
		Value: apiAddressFlag.Value,
	}

	accessPolicyAddressFlag = cli.StringFlag{
		Name:  "access-policy-address",
		Usage: "URL of trust oracle endpoint for retrieving lists of access policies",
		Value: metadata.DefaultNetwork.AccessPolicyOracleAddress,
	}

	brokerAddressFlag = cli.StringFlag{
		Name:  "broker-address",
		Usage: "URI of message broker",
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

	natPunchingFlag = cli.BoolTFlag{
		Name:  "experiment-natpunching",
		Usage: "Enables experimental NAT hole punching",
	}
)

// RegisterFlagsNetwork function register network flags to flag list
func RegisterFlagsNetwork(flags *[]cli.Flag) {
	*flags = append(
		*flags,
		testFlag, localnetFlag,
		identityCheckFlag,
		natPunchingFlag,
		apiAddressFlag, apiAddressFlagDepreciated,
		brokerAddressFlag,
		etherRPCFlag, etherContractPaymentsFlag,
		qualityOracleFlag, accessPolicyAddressFlag,
	)
}

// ParseFlagsNetwork function fills in directory options from CLI context
func ParseFlagsNetwork(ctx *cli.Context) node.OptionsNetwork {
	return node.OptionsNetwork{
		Testnet:  ctx.GlobalBool(testFlag.Name),
		Localnet: ctx.GlobalBool(localnetFlag.Name),

		ExperimentIdentityCheck: ctx.GlobalBool(identityCheckFlag.Name),
		ExperimentNATPunching:   ctx.GlobalBool(natPunchingFlag.Name),

		MysteriumAPIAddress:         ctx.GlobalString(apiAddressFlag.Name),
		AccessPolicyEndpointAddress: ctx.GlobalString(accessPolicyAddressFlag.Name),
		BrokerAddress:               ctx.GlobalString(brokerAddressFlag.Name),

		EtherClientRPC:       ctx.GlobalString(etherRPCFlag.Name),
		EtherPaymentsAddress: ctx.GlobalString(etherContractPaymentsFlag.Name),

		QualityOracle: ctx.GlobalString(qualityOracleFlag.Name),
	}
}
