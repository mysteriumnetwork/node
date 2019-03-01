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

package service

import (
	"encoding/json"

	"github.com/mysteriumnetwork/node/core/service"
	"github.com/urfave/cli"
)

// Options describes options which are required to start Wireguard service
type Options struct {
	ConnectDelay int `json:"connectDelay"`
}

var (
	delayFlag = cli.IntFlag{
		Name:  "connect.delay",
		Usage: "Consumer is delayed by specified time (2000 millisec default) if provider is behind NAT",
		Value: 2000,
	}
)

// RegisterFlags function register Wireguard flags to flag list
func RegisterFlags(flags *[]cli.Flag) {
	*flags = append(*flags, delayFlag)
}

// ParseFlags function fills in Wireguard options from CLI context
func ParseFlags(ctx *cli.Context) service.Options {
	return Options{
		ConnectDelay: ctx.Int(delayFlag.Name),
	}
}

// ParseJSONOptions function fills in Openvpn options from JSON request
func ParseJSONOptions(request *json.RawMessage) (service.Options, error) {
	if request == nil {
		return Options{}, nil
	}

	var opts Options
	err := json.Unmarshal(*request, &opts)
	return opts, err
}
