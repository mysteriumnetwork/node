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
	"fmt"

	"github.com/mysteriumnetwork/node/core/node"
	"github.com/urfave/cli"
)

var (
	qualityTypeFlag = cli.StringFlag{
		Name: "quality.type",
		Usage: fmt.Sprintf(
			"Quality Oracle adapter. Options:  (%s, %s - %s)",
			node.QualityTypeMORQA,
			node.QualityTypeNone,
			"opt-out from sending quality metrics",
		),
		Value: string(node.QualityTypeMORQA),
	}
	qualityAddressFlag = cli.StringFlag{
		Name: "quality.address",
		Usage: fmt.Sprintf(
			"Address of specific Quality Oracle adapter given in '--%s'",
			qualityTypeFlag.Name,
		),
		Value: "http://metrics.mysterium.network:8091",
	}
)

// RegisterFlagsQuality function register Quality Oracle flags to flag list
func RegisterFlagsQuality(flags *[]cli.Flag) {
	*flags = append(*flags, qualityTypeFlag, qualityAddressFlag)
}

// ParseFlagsQuality function fills in Quality Oracle from CLI context
func ParseFlagsQuality(ctx *cli.Context) node.OptionsQuality {
	return node.OptionsQuality{
		Type:    node.QualityType(ctx.GlobalString(qualityTypeFlag.Name)),
		Address: ctx.GlobalString(qualityAddressFlag.Name),
	}
}
