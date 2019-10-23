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
	"gopkg.in/urfave/cli.v1"
)

var (
	maxAccountantFee = cli.IntFlag{
		Name:  "payments.accountant.max.fee",
		Value: 1500,
		Usage: "The max fee that we'll accept from an accountant. In percentiles. 1500 means 15%",
	}
	bcTimeout = cli.DurationFlag{
		Name:  "payments.bc.timeout",
		Value: time.Second * 30,
		Usage: "The duration we'll wait before timing out BC calls.",
	}
)

// RegisterFlagsPayments registers flags to control payments
func RegisterFlagsPayments(flags *[]cli.Flag) {
	*flags = append(*flags, maxAccountantFee)
}

// ParsePaymentFlags parses registered flags and puts them into options structure
func ParsePaymentFlags(ctx *cli.Context) node.OptionsPayments {
	return node.OptionsPayments{
		MaxAllowedPaymentPercentile: ctx.GlobalInt(maxAccountantFee.Name),
		BCTimeout:                   ctx.GlobalDuration(bcTimeout.Name),
	}
}
