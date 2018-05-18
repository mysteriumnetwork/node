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

package openvpn

type ClientConfig struct {
	*Config
}

func (c *ClientConfig) SetClientMode(serverIP string, serverPort int) {
	c.setFlag("client")
	c.setParam("script-security", "2")
	c.setFlag("auth-nocache")
	c.setParam("remote", serverIP)
	c.SetPort(serverPort)
	c.setFlag("nobind")
	c.setParam("remote-cert-tls", "server")
	c.setFlag("auth-user-pass")
	c.setFlag("management-query-passwords")
}

func (c *ClientConfig) SetProtocol(protocol string) {
	if protocol == "tcp" {
		c.setParam("proto", "tcp-client")
	} else if protocol == "udp" {
		c.setFlag("explicit-exit-notify")
	}
}
