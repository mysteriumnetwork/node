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

package connection

import (
	"encoding/json"

	wg "github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/services/wireguard/endpoint"
	"github.com/mysteriumnetwork/node/session"
)

// WireguardAckHandler handles the acknowledgements on session creation for wireguard
// It takes gets the service side config from sesion create
func WireguardAckHandler(sessionResponse session.SessionDto, ackSend func(payload interface{}) error) (json.RawMessage, error) {
	privateKey, err := endpoint.GeneratePrivateKey()
	if err != nil {
		return sessionResponse.Config, err
	}
	publicKey, err := endpoint.PrivateKeyToPublicKey(privateKey)
	if err != nil {
		return sessionResponse.Config, err
	}

	err = ackSend(wg.ConsumerPublicKey{PublicKey: publicKey})
	if err != nil {
		return sessionResponse.Config, err
	}

	parsed := &wg.ServiceConfig{}
	err = json.Unmarshal(sessionResponse.Config, parsed)
	if err != nil {
		return sessionResponse.Config, err
	}
	parsed.Consumer.PrivateKey = privateKey

	res, err := json.Marshal(parsed)
	return res, err
}
