// +build linux,!android

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

import (
	"encoding/base64"

	"github.com/mdlayher/wireguardctrl/wgtypes"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint/kernelspace"
	"github.com/mysteriumnetwork/node/services/wireguard/resources"
)

// NewConnectionEndpoint creates new wireguard connection endpoint.
func NewConnectionEndpoint(ipResolver ip.Resolver) (*connectionEndpoint, error) {
	wgClient, err := kernelspace.NewWireguardClient()
	if err != nil {
		return nil, err
	}

	return &connectionEndpoint{
		wgClient:          wgClient,
		ipResolver:        ipResolver,
		resourceAllocator: resources.Allocator{},
	}, nil
}

// GeneratePrivateKey creates new wireguard private key
func GeneratePrivateKey() (string, error) {
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return "", err
	}
	return key.String(), nil
}

// PrivateKeyToPublicKey generates wireguard public key from private key
func PrivateKeyToPublicKey(key string) (string, error) {
	k, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", err
	}
	privateKey, err := wgtypes.NewKey(k)
	if err != nil {
		return "", err
	}
	return privateKey.PublicKey().String(), nil
}
