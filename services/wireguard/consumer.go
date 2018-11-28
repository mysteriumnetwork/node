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

package wireguard

import (
	"errors"
	"net"

	"github.com/mdlayher/wireguardctrl/wgtypes"
)

var errNoIPAddress = errors.New("IP address required")

type consumer struct {
	ip         net.IPNet
	peer       wgtypes.PeerConfig
	privateKey wgtypes.Key
}

func (c consumer) Config() (serviceConfig, error) {
	if len(c.peer.AllowedIPs) == 0 {
		return serviceConfig{}, errNoIPAddress
	}

	var config serviceConfig

	config.Provider.IP = c.peer.AllowedIPs[0]
	config.Provider.Endpoint = *c.peer.Endpoint
	config.Provider.PublicKey = c.peer.PublicKey

	config.Consumer.IP = c.ip
	config.Consumer.PrivateKey = c.privateKey

	return config, nil
}
