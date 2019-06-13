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

package auth

import (
	"encoding/base64"
	"errors"
	"strings"
)

// Auth provides an authentication method for builtin UI.
type Auth struct {
	storage Storage
}

// NewAuth creates an authenticator
func NewAuth(storage Storage) *Auth {
	return &Auth{
		storage: storage,
	}
}

// AuthenticateHTTPBasic authenticates user using http basic auth header
func (a *Auth) AuthenticateHTTPBasic(header string) error {
	username, password, err := parseBasicAuthHeader(header)
	if err != nil {
		return err
	}
	return NewCredentials(username, password, a.storage).Validate()
}

func parseBasicAuthHeader(header string) (user, password string, err error) {
	header = strings.TrimPrefix(header, "Basic ")
	out, err := base64.StdEncoding.DecodeString(header)
	if err != nil {
		return "", "", err
	}

	cred := strings.Split(string(out), ":")
	if len(cred) != 2 {
		return "", "", errors.New("incorrect basic auth header")
	}

	return cred[0], cred[1], nil
}
