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
	"flag"
	"testing"

	"github.com/mysteriumnetwork/node/config"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

var DefaultOptionsOpenvpn = Options{
	Protocol: config.FlagOpenvpnProtocol.Value,
	Port:     config.FlagOpenvpnPort.Value,
	Subnet:   config.FlagOpenvpnSubnet.Value,
	Netmask:  config.FlagOpenvpnNetmask.Value,
}

func Test_ParseJSONOptions_HandlesNil(t *testing.T) {
	configureDefaults()
	options, err := ParseJSONOptions(nil)

	assert.NoError(t, err)
	assert.Equal(t, DefaultOptionsOpenvpn, options)
}

func Test_ParseJSONOptions_HandlesEmptyRequest(t *testing.T) {
	configureDefaults()
	request := json.RawMessage(`{}`)
	options, err := ParseJSONOptions(&request)

	assert.NoError(t, err)
	assert.Equal(t, DefaultOptionsOpenvpn, options)
}

func Test_ParseJSONOptions_ValidRequest(t *testing.T) {
	configureDefaults()
	request := json.RawMessage(`{"port": 1123, "protocol": "udp", "subnet": "10.10.10.0", "netmask": "255.255.255.0"}`)
	options, err := ParseJSONOptions(&request)

	assert.NoError(t, err)
	assert.Equal(t, Options{
		Protocol: "udp",
		Port:     1123,
		Subnet:   "10.10.10.0",
		Netmask:  "255.255.255.0",
	}, options)
}

func configureDefaults() {
	ctx := emptyContext()
	config.ParseFlagsServiceOpenvpn(ctx)
}

func emptyContext() *cli.Context {
	return cli.NewContext(nil, flag.NewFlagSet("", flag.ContinueOnError), nil)
}
