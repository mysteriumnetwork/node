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
	"gopkg.in/urfave/cli.v1/altsrc"
)

var (
	testFlag = altsrc.NewBoolFlag(cli.BoolFlag{
		Name:  "testnet",
		Usage: "Defines test network configuration",
	})
	localnetFlag = altsrc.NewBoolFlag(cli.BoolFlag{
		Name:  "localnet",
		Usage: "Defines network configuration which expects locally deployed broker and discovery services",
	})

	identityCheckFlag = altsrc.NewBoolFlag(cli.BoolFlag{
		Name:  "experiment-identity-check",
		Usage: "Enables experimental identity check",
	})

	apiAddressFlag = altsrc.NewStringFlag(cli.StringFlag{
		Name:  "api.address",
		Usage: "URL of Mysterium API",
		Value: metadata.DefaultNetwork.MysteriumAPIAddress,
	})
	apiAddressFlagDeprecated = altsrc.NewStringFlag(cli.StringFlag{
		Name:  "discovery-address",
		Usage: fmt.Sprintf("URL of Mysterium API (DEPRECATED, start using '--%s')", apiAddressFlag.Name),
		Value: apiAddressFlag.Value,
	})

	accessPolicyAddressFlag = altsrc.NewStringFlag(cli.StringFlag{
		Name:  "access-policy-address",
		Usage: "URL of trust oracle endpoint for retrieving lists of access policies",
		Value: metadata.DefaultNetwork.AccessPolicyOracleAddress,
	})

	mmnAddressFlag = altsrc.NewStringFlag(cli.StringFlag{
		Name:  "mmn-address",
		Usage: "URL of my.mysterium.network API",
		Value: metadata.DefaultNetwork.MMNAddress,
	})

	brokerAddressFlag = altsrc.NewStringFlag(cli.StringFlag{
		Name:  "broker-address",
		Usage: "URI of message broker",
		Value: metadata.DefaultNetwork.BrokerAddress,
	})

	etherRPCFlag = altsrc.NewStringFlag(cli.StringFlag{
		Name:  "ether.client.rpc",
		Usage: "URL or IPC socket to connect to ethereum node, anything what ethereum client accepts - works",
		Value: metadata.DefaultNetwork.EtherClientRPC,
	})
	etherContractPaymentsFlag = altsrc.NewStringFlag(cli.StringFlag{
		Name:  "ether.contract.payments",
		Usage: "Address of payments contract",
		Value: metadata.DefaultNetwork.PaymentsContractAddress.String(),
	})

	qualityOracleFlag = altsrc.NewStringFlag(cli.StringFlag{
		Name:  "quality-oracle.address",
		Usage: "Address of the quality oracle service",
		Value: metadata.DefaultNetwork.QualityOracle,
	})

	natPunchingFlag = altsrc.NewBoolTFlag(cli.BoolTFlag{
		Name:  "experiment-natpunching",
		Usage: "Enables experimental NAT hole punching",
	})
)

// RegisterFlagsNetwork function register network flags to flag list
func RegisterFlagsNetwork(flags *[]cli.Flag) {
	*flags = append(
		*flags,
		testFlag, localnetFlag,
		identityCheckFlag,
		natPunchingFlag,
		apiAddressFlag, apiAddressFlagDeprecated,
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
