/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/config/urfavecli/cliflags"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/rs/zerolog/log"
	"gopkg.in/urfave/cli.v1"
)

// Options describes options which are required to start Openvpn service
type Options struct {
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
	Subnet   string `json:"subnet"`
	Netmask  string `json:"netmask"`
}

var (
	protocolFlag = cli.StringFlag{
		Name:  "openvpn.proto",
		Usage: "OpenVPN protocol to use. Options: { udp, tcp }",
	}
	portFlag = cli.IntFlag{
		Name:  "openvpn.port",
		Usage: "OpenVPN port to use. If not specified, random port will be used",
	}
	subnetFlag = cli.StringFlag{
		Name:  "openvpn.subnet",
		Usage: "OpenVPN subnet that will be used to connecting VPN clients",
	}
	netmaskFlag = cli.StringFlag{
		Name:  "openvpn.netmask",
		Usage: "OpenVPN subnet netmask ",
	}
	defaultOptions = Options{
		Protocol: "udp",
		Port:     0,
		Subnet:   "10.8.0.0",
		Netmask:  "255.255.255.0",
	}
)

// RegisterFlags registers OpenVPN CLI flags for parsing them later
func RegisterFlags(flags *[]cli.Flag) {
	*flags = append(*flags, protocolFlag, portFlag, subnetFlag, netmaskFlag)
}

// Configure parses CLI flags and registers value to configuration
func Configure(ctx *cli.Context) {
	configureDefaults()
	configureCLI(ctx)
}

func configureDefaults() {
	config.Current.SetDefault(protocolFlag.Name, defaultOptions.Protocol)
	config.Current.SetDefault(portFlag.Name, defaultOptions.Port)
	config.Current.SetDefault(subnetFlag.Name, defaultOptions.Subnet)
	config.Current.SetDefault(netmaskFlag.Name, defaultOptions.Netmask)
}

func configureCLI(ctx *cli.Context) {
	cliflags.SetString(config.Current, protocolFlag.Name, ctx)
	cliflags.SetInt(config.Current, portFlag.Name, ctx)
	cliflags.SetString(config.Current, subnetFlag.Name, ctx)
	cliflags.SetString(config.Current, netmaskFlag.Name, ctx)
}

// ConfiguredOptions returns effective OpenVPN service options from configuration
func ConfiguredOptions() Options {
	return Options{
		Protocol: config.Current.GetString(protocolFlag.Name),
		Port:     config.Current.GetInt(portFlag.Name),
		Subnet:   config.Current.GetString(subnetFlag.Name),
		Netmask:  config.Current.GetString(netmaskFlag.Name),
	}
}

// ParseJSONOptions function fills in OpenVPN options from JSON request, falling back to configured options for
// missing values
func ParseJSONOptions(request *json.RawMessage) (service.Options, error) {
	var requestOptions = ConfiguredOptions()
	if request == nil {
		return requestOptions, nil
	}
	err := json.Unmarshal(*request, &requestOptions)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse options from request, using effective options")
		return &Options{}, err
	}
	return requestOptions, nil
}
