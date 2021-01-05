/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package clio

import (
	"fmt"

	"github.com/mysteriumnetwork/node/config"
	tequilapi_client "github.com/mysteriumnetwork/node/tequilapi/client"

	"github.com/urfave/cli/v2"
)

// NewTequilApiClient - initializes and returns a pointer to tequilapi client - also fetches config using it
func NewTequilApiClient(ctx *cli.Context) (*tequilapi_client.Client, error) {
	address := TequilAPIAddress(ctx)
	port := TequilAPIPort(ctx)
	client := tequilapi_client.NewClient(address, port)

	_, err := client.Healthcheck()
	if err != nil {
		Error(fmt.Sprintf("failed to connect to node via url: %s:%d", address, port))
		return nil, err
	}
	return client, nil
}

// TequilAPIAddress - wil resolve default tequilapi address or from flag if one is provided
func TequilAPIAddress(ctx *cli.Context) string {
	flag := config.FlagTequilapiAddress

	if ctx.IsSet(flag.Name) {
		return ctx.String(flag.Name)
	}

	return flag.Value
}

// TequilAPIPort - wil resolve default tequilapi port or from flag if one is provided
func TequilAPIPort(ctx *cli.Context) int {
	flag := config.FlagTequilapiPort

	if ctx.IsSet(flag.Name) {
		return ctx.Int(flag.Name)
	}

	return flag.Value
}
