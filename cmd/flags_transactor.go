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

package cmd

import (
	"time"

	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/metadata"
	"gopkg.in/urfave/cli.v1"
)

var (
	transactorAddressFlag = cli.StringFlag{
		Name:  "transactor.address",
		Usage: "transactor URL address",
		Value: metadata.DefaultNetwork.TransactorAddress,
	}
	registryAddressFlag = cli.StringFlag{
		Name:  "transactor.registry-address",
		Usage: "registry contract address used to register identity",
		Value: metadata.DefaultNetwork.RegistryAddress,
	}
	channelImplementationFlag = cli.StringFlag{
		Name:  "transactor.channel-implementation",
		Usage: "channel implementation address",
		Value: metadata.DefaultNetwork.ChannelImplAddress,
	}
	providerMaxRegistrationAttemptsFlag = cli.IntFlag{
		Name:  "transactor.provider.max-registration-attempts",
		Usage: "the max attempts the provider will make to register before giving up",
		Value: 10,
	}
	providerRegistrationRetryDelay = cli.DurationFlag{
		Name:  "transactor.provider.registration-retry-delay",
		Usage: "the duration that the provider will wait between each retry",
		Value: time.Minute * 3,
	}
	providerRegistrationStake = cli.Uint64Flag{
		Name:  "transactor.provider.registration-stake",
		Usage: "the stake we'll use when registering provider",
		Value: 125000000000,
	}
)

// RegisterFlagsTransactor function register network flags to flag list
func RegisterFlagsTransactor(flags *[]cli.Flag) {
	*flags = append(
		*flags,
		transactorAddressFlag,
		registryAddressFlag,
		channelImplementationFlag,
		providerMaxRegistrationAttemptsFlag,
		providerRegistrationRetryDelay,
		providerRegistrationStake,
	)
}

// ParseFlagsTransactor function fills in transactor options from CLI context
func ParseFlagsTransactor(ctx *cli.Context) node.OptionsTransactor {
	return node.OptionsTransactor{
		TransactorEndpointAddress:       ctx.GlobalString(transactorAddressFlag.Name),
		RegistryAddress:                 ctx.GlobalString(registryAddressFlag.Name),
		ChannelImplementation:           ctx.GlobalString(channelImplementationFlag.Name),
		ProviderMaxRegistrationAttempts: ctx.GlobalInt(providerMaxRegistrationAttemptsFlag.Name),
		ProviderRegistrationRetryDelay:  ctx.GlobalDuration(providerRegistrationRetryDelay.Name),
		ProviderRegistrationStake:       ctx.GlobalUint64(providerRegistrationStake.Name),
	}
}
