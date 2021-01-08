/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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
)

// Storage for Credentials.
type Storage interface {
	GetValue(bucket string, key interface{}, to interface{}) error
	SetValue(bucket string, key interface{}, to interface{}) error
}

const credentialsDBBucket = "app-credentials"

// MigrateCredentials check if a password file exists, if it doesn't
// it will create it following the rules:
//
// Create a new file with a password from testnet2 Storage if
// testnet2 doesn't have a password set, else use testnet.
//
// If password can't be found in either Storage and no file exists
// we assume that the user is new and he will create it.
func MigrateCredentials(passStorages []Storage) error {
	handler := NewCredentialsManager(config.GetString(config.FlagDataDir))
	_, err := handler.getPassword()
	if err == nil {
		// Password in a file exists, we can exit.
		return nil
	}

	for _, s := range passStorages {
		var storedHash string
		err := s.GetValue(
			credentialsDBBucket,
			config.FlagTequilapiUsername.Value,
			&storedHash,
		)

		if err == nil {
			return handler.setPassword(storedHash)
		}
	}

	// Password not found in any of the given storages
	// we can assume that user is launching the app for the first time.
	// Just continue as normal.
	return nil
}
