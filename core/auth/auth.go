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
	"github.com/mysteriumnetwork/node/config"
	"github.com/rs/zerolog/log"
)

// Authenticator wraps CredentialsManager to provide
// an easy way of authentication for builtin UI.
type Authenticator struct {
	manager *CredentialsManager
}

// NewAuthenticator creates an authenticator.
func NewAuthenticator() *Authenticator {
	pswDir := config.GetString(config.FlagDataDir)
	return &Authenticator{
		manager: NewCredentialsManager(pswDir),
	}
}

// CheckCredentials checks if provided username and password combo is valid
// comparing it to stored credentials.
func (a *Authenticator) CheckCredentials(username, password string) error {
	return a.manager.Validate(username, password)
}

// ChangePassword changes user password.
func (a *Authenticator) ChangePassword(username, oldPassword, newPassword string) error {
	err := a.manager.Validate(username, oldPassword)
	if err != nil {
		log.Info().Err(err).Msg("Bad credentials for changing password")
		return ErrUnauthorized
	}
	err = a.manager.SetPassword(newPassword)
	if err != nil {
		log.Info().Err(err).Msg("Error changing password")
		return err
	}
	log.Info().Msgf("%q user password changed successfully", username)
	return nil
}
