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
	"strings"

	"github.com/urfave/cli/v2"
)

var (
	// FlagIdentity keystore's identity.
	FlagIdentity = cli.StringFlag{
		Name:  "identity",
		Usage: "Keystore's identity used to provide service. If not given identity will be created automatically",
		Value: "",
	}
	// FlagIdentityPassphrase passphrase to unlock the identity.
	FlagIdentityPassphrase = cli.StringFlag{
		Name:  "identity.passphrase",
		Usage: "Used to unlock keystore's identity",
		Value: "",
	}

	// FlagAgreedTermsConditions agree with terms & conditions.
	FlagAgreedTermsConditions = cli.BoolFlag{
		Name:  "agreed-terms-and-conditions",
		Usage: "Agree with terms & conditions for consumer, provider or both depending on the command executed",
	}

	// FlagAccessPolicyList a comma-separated list of access policies that determines allowed identities to use the service.
	FlagAccessPolicyList = cli.StringFlag{
		Name:  "access-policy.list",
		Usage: "Comma separated list that determines the access policies applied to provide service.",
		Value: "",
	}

	// FlagPaymentPriceGiB sets the price/GiB to provided service.
	FlagPaymentPriceGiB = cli.Float64Flag{
		Name:  "payment.price-gib",
		Usage: "Sets the price/GiB applied to provider service.",
		Value: 0.1,
	}
	// FlagPaymentPriceHour sets the price/hour to provided service.
	FlagPaymentPriceHour = cli.Float64Flag{
		Name:  "payment.price-hour",
		Usage: "Sets the price/hour applied to provider service.",
		Value: 0.00006,
	}

	// FlagActiveServices a comma-separated list of active services.
	FlagActiveServices = cli.StringFlag{
		Name:  "active-services",
		Usage: "Comma separated list of active services.",
		Value: strings.Join([]string{"scraping", "data_transfer", "dvpn"}, ","),
	}

	// FlagStoppedServices a comma-separated list of stopped services.
	FlagStoppedServices = cli.StringFlag{
		Name:   "stopped-services",
		Usage:  "Comma separated list of stopped services.",
		Value:  strings.Join([]string{}, ","),
		Hidden: true,
	}
)

// RegisterFlagsServiceStart registers CLI flags used to start a service.
func RegisterFlagsServiceStart(flags *[]cli.Flag) {
	*flags = append(*flags,
		&FlagIdentity,
		&FlagIdentityPassphrase,
		&FlagAgreedTermsConditions,
		&FlagPaymentPriceGiB,
		&FlagPaymentPriceHour,
		&FlagAccessPolicyList,
		&FlagActiveServices,
		&FlagStoppedServices,
	)
}

// ParseFlagsServiceStart parses service start CLI flags and registers values to the configuration
func ParseFlagsServiceStart(ctx *cli.Context) {
	Current.ParseStringFlag(ctx, FlagIdentity)
	Current.ParseStringFlag(ctx, FlagIdentityPassphrase)
	Current.ParseBoolFlag(ctx, FlagAgreedTermsConditions)
	Current.ParseFloat64Flag(ctx, FlagPaymentPriceGiB)
	Current.ParseFloat64Flag(ctx, FlagPaymentPriceHour)
	Current.ParseStringFlag(ctx, FlagAccessPolicyList)
	Current.ParseStringFlag(ctx, FlagActiveServices)
	Current.ParseStringFlag(ctx, FlagStoppedServices)
}
