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

import "github.com/rs/zerolog/log"

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

// CheckCredentials authenticates user by password
func (a *Authenticator) CheckCredentials(username, password string) error {
	return NewCredentials(username, password, a.storage).Validate()
}

// ChangePassword changes user password
func (a *Authenticator) ChangePassword(username, oldPassword, newPassword string) (err error) {
	err = NewCredentials(username, oldPassword, a.storage).Validate()
	if err != nil {
		log.Info().Err(err).Msg("Bad credentials for changing password")
		return ErrUnauthorized
	}
	err = NewCredentials(username, newPassword, a.storage).Set()
	if err != nil {
		log.Info().Err(err).Msg("Error changing password")
		return err
	}
	log.Info().Msgf("%q user password changed successfully", username)
	return nil
}
