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

	log "github.com/cihub/seelog"
)

// Authenticator provides an authentication method for builtin UI.
type Authenticator struct {
	storage Storage
}

// NewAuthenticator creates an authenticator
func NewAuthenticator(storage Storage) *Authenticator {
	return &Authenticator{
		storage: storage,
	}
}

// AuthenticateHTTPBasic authenticates user using http basic auth header
func (a *Authenticator) AuthenticateHTTPBasic(header string) error {
	username, password, err := parseBasicAuthHeader(header)
	if err != nil {
		return err
	}
	return NewCredentials(username, password, a.storage).Validate()
}

// ChangePassword changes user password
func (a *Authenticator) ChangePassword(username, oldPassword, newPassword string) (err error) {
	err = NewCredentials(username, oldPassword, a.storage).Validate()
	if err != nil {
		log.Info("bad credentials for changing password: ", err)
		return ErrUnauthorized
	}
	err = NewCredentials(username, newPassword, a.storage).Set()
	if err != nil {
		log.Infof("error changing password: ", err)
		return err
	}
	log.Infof("%s user password changed successfully", username)
	return nil
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
