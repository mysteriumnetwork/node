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

	"gopkg.in/urfave/cli.v1"
)

var (
	// FlagPaymentsMaxAccountantFee represents the max accountant fee.
	FlagPaymentsMaxAccountantFee = cli.IntFlag{
		Name:  "payments.accountant.max.fee",
		Value: 1500,
		Usage: "The max fee that we'll accept from an accountant. In percentiles. 1500 means 15%",
	}
	// FlagPaymentsBCTimeout represents the BC call timeout.
	FlagPaymentsBCTimeout = cli.DurationFlag{
		Name:  "payments.bc.timeout",
		Value: time.Second * 30,
		Usage: "The duration we'll wait before timing out BC calls.",
	}
	// FlagPaymentsAccountantPromiseSettleThreshold represents the percentage of balance left when we go for promise settling.
	FlagPaymentsAccountantPromiseSettleThreshold = cli.Float64Flag{
		Name:  "payments.accountant.promise.threshold",
		Value: 0.1,
		Usage: "The percentage of balance before we settle promises",
	}
	// FlagPaymentsAccountantPromiseSettleTimeout represents the time we wait for confirmation of the promise settlement.
	FlagPaymentsAccountantPromiseSettleTimeout = cli.DurationFlag{
		Name:  "payments.accountant.promise.timeout",
		Value: time.Hour * 2,
		Usage: "The duration we'll wait before timing out our wait for promise settle.",
	}
	// FlagPaymentsMystSCAddress represents the myst smart contract address
	FlagPaymentsMystSCAddress = cli.StringFlag{
		Name:  "payments.mystscaddress",
		Value: "0xe67e41367c1e17ede951a528b2a8be35c288c787",
		Usage: "The address of myst token smart contract",
	}
	// FlagPaymentsMaxRRecovery represents the max r recovery.
	FlagPaymentsMaxRRecovery = cli.Uint64Flag{
		Name:  "payments.max.R.Recovery",
		Value: 150,
		Usage: "The max number of invoices we'll go through and reveal R's in case of a dispute with the accountant",
	}
	// FlagPaymentsDisable disables the payment system
	FlagPaymentsDisable = cli.BoolFlag{
		Name:  "payments.disable",
		Usage: "Disables payments and moves to a backwards compatible legacy mode",
	}
)

// RegisterFlagsPayments function register payments flags to flag list.
func RegisterFlagsPayments(flags *[]cli.Flag) {
	*flags = append(
		*flags,
		FlagPaymentsMaxAccountantFee,
		FlagPaymentsBCTimeout,
		FlagPaymentsAccountantPromiseSettleThreshold,
		FlagPaymentsAccountantPromiseSettleTimeout,
		FlagPaymentsMystSCAddress,
		FlagPaymentsMaxRRecovery,
		FlagPaymentsDisable,
	)
}

// ParseFlagsPayments function fills in payments options from CLI context.
func ParseFlagsPayments(ctx *cli.Context) {
	Current.ParseIntFlag(ctx, FlagPaymentsMaxAccountantFee)
	Current.ParseDurationFlag(ctx, FlagPaymentsBCTimeout)
	Current.ParseFloat64Flag(ctx, FlagPaymentsAccountantPromiseSettleThreshold)
	Current.ParseDurationFlag(ctx, FlagPaymentsAccountantPromiseSettleTimeout)
	Current.ParseStringFlag(ctx, FlagPaymentsMystSCAddress)
	Current.ParseUInt64Flag(ctx, FlagPaymentsMaxRRecovery)
	Current.ParseBoolFlag(ctx, FlagPaymentsDisable)
}
