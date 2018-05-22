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

package credentials

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn/middlewares/client/auth"
	ovpnsession "github.com/mysterium/node/openvpn/session"
	"github.com/mysterium/node/session"
)

// SignatureCredentialsProvider returns session id as username and id signed with given signer as password
func SignatureCredentialsProvider(id session.SessionID, signer identity.Signer) auth.CredentialsProvider {
	return func() (string, string, error) {
		signature, err := signer.Sign([]byte(ovpnsession.SignaturePrefix + id))
		return string(id), signature.Base64(), err
	}
}
