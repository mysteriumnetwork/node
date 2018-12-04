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

package endpoint

import "net"

type device struct {
	name          *string
	publicKey     *string
	privateKey    *string
	listenPort    *int
	peerEndpoint  *net.UDPAddr
	peerPublicKey *string
}

func (d device) Name() *string {
	return d.name
}

func (d device) PublicKey() *string {
	return d.publicKey
}

func (d device) PrivateKey() *string {
	return d.privateKey
}

func (d device) ListenPort() *int {
	return d.listenPort
}

func (d device) PeerEndpoint() *net.UDPAddr {
	return d.peerEndpoint
}
func (d device) PeerPublicKey() *string {
	return d.peerPublicKey
}
