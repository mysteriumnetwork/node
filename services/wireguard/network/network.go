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

package network

import (
	"errors"

	"github.com/mdlayher/wireguardctrl"
)

// Provider is a configuration data required for establishing connection to the service provider.
type Provider struct {
	PublicKey string
	Endpoint  string
}

// Consumer is a configuration data required to configure service consumer.
type Consumer struct {
	PrivateKey string // TODO peer private key should be generated on consumer side
	IP         string
}

var errNotSupported = errors.New("OS is not supported for wireguard transport")

type network struct {
	name     string
	publicIP string
	wgClient *wireguardctrl.Client
}
