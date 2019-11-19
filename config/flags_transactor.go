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
	"time"

	"github.com/mysteriumnetwork/node/metadata"
	"gopkg.in/urfave/cli.v1"
)

var (
	// FlagTransactorAddress transactor URL.
	FlagTransactorAddress = cli.StringFlag{
		Name:  "transactor.address",
		Usage: "Transactor URL address",
		Value: metadata.DefaultNetwork.TransactorAddress,
	}
	// FlagTransactorRegistryAddress registry contract address used for identity registration.
	FlagTransactorRegistryAddress = cli.StringFlag{
		Name:  "transactor.registry-address",
		Usage: "Registry contract address used to register identity",
		Value: metadata.DefaultNetwork.RegistryAddress,
	}
	// FlagTransactorChannelImplementation the channel implementation sc address used for channel address calculation.
	FlagTransactorChannelImplementation = cli.StringFlag{
		Name:  "transactor.channel-implementation",
		Usage: "channel implementation address",
		Value: metadata.DefaultNetwork.ChannelImplAddress,
	}
	// FlagTransactorProviderMaxRegistrationAttempts determines the number of registration attempts that the provider will attempt before giving up.
	FlagTransactorProviderMaxRegistrationAttempts = cli.IntFlag{
		Name:  "transactor.provider.max-registration-attempts",
		Usage: "the max attempts the provider will make to register before giving up",
		Value: 10,
	}
	// FlagTransactorProviderRegistrationRetryDelay determines the delay between each provider registration attempts.
	FlagTransactorProviderRegistrationRetryDelay = cli.DurationFlag{
		Name:  "transactor.provider.registration-retry-delay",
		Usage: "the duration that the provider will wait between each retry",
		Value: time.Minute * 3,
	}
	// FlagTransactorProviderRegistrationStake determines the stake size for registration of provider
	FlagTransactorProviderRegistrationStake = cli.Uint64Flag{
		Name:  "transactor.provider.registration-stake",
		Usage: "the stake we'll use when registering provider",
		Value: 125000000000,
	}
)

// RegisterFlagsTransactor function register network flags to flag list
func RegisterFlagsTransactor(flags *[]cli.Flag) {
	*flags = append(
		*flags,
		FlagTransactorAddress,
		FlagTransactorRegistryAddress,
		FlagTransactorChannelImplementation,
		FlagTransactorProviderMaxRegistrationAttempts,
		FlagTransactorProviderRegistrationRetryDelay,
		FlagTransactorProviderRegistrationStake,
	)
}

// ParseFlagsTransactor function fills in transactor options from CLI context
func ParseFlagsTransactor(ctx *cli.Context) {
	Current.ParseStringFlag(ctx, FlagTransactorAddress)
	Current.ParseStringFlag(ctx, FlagTransactorChannelImplementation)
	Current.ParseStringFlag(ctx, FlagTransactorRegistryAddress)
	Current.ParseIntFlag(ctx, FlagTransactorProviderMaxRegistrationAttempts)
	Current.ParseDurationFlag(ctx, FlagTransactorProviderRegistrationRetryDelay)
	Current.ParseUInt64Flag(ctx, FlagTransactorProviderRegistrationStake)
}
