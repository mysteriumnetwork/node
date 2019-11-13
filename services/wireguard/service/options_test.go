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
	"net"
	"testing"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/stretchr/testify/assert"
	"gopkg.in/urfave/cli.v1"
)

func Test_ParseJSONOptions_HandlesNil(t *testing.T) {
	configureDefaults()
	options, err := ParseJSONOptions(nil)

	assert.NoError(t, err)
	assert.Equal(t, DefaultOptions, options)
}

func Test_ParseJSONOptions_HandlesEmptyRequest(t *testing.T) {
	configureDefaults()
	request := json.RawMessage(`{}`)
	options, err := ParseJSONOptions(&request)

	assert.NoError(t, err)
	assert.Equal(t, DefaultOptions, options)
}

func Test_ParseJSONOptions_ValidRequest(t *testing.T) {
	configureDefaults()
	request := json.RawMessage(`{"connectDelay": 3000, "ports": "52820:53075", "subnet":"10.10.0.0/16"}`)
	options, err := ParseJSONOptions(&request)

	assert.NoError(t, err)
	assert.Equal(t, Options{
		ConnectDelay: 3000,
		Ports:        &port.Range{Start: 52820, End: 53075},
		Subnet: net.IPNet{
			IP:   net.ParseIP("10.10.0.0").To4(),
			Mask: net.IPv4Mask(255, 255, 0, 0),
		},
	}, options)
}

func configureDefaults() {
	ctx := emptyContext()
	config.ParseFlagsServiceWireguard(ctx)
}

func emptyContext() *cli.Context {
	return cli.NewContext(nil, flag.NewFlagSet("", flag.ContinueOnError), nil)
}
