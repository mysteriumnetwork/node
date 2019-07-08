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
	"crypto/rand"

	"github.com/pkg/errors"
)

const encryptionKeyBucket = "jwt"
const encryptionKeyName = "jwt-encryption-key"

// NewJWTEncryptionKey creates and stores or re-uses an existing JWT encryption key
func NewJWTEncryptionKey(storage Storage) (JWTEncryptionKey, error) {
	key := JWTEncryptionKey{}
	err := storage.GetValue(encryptionKeyBucket, encryptionKeyName, &key)
	if err != nil {
		key, err = generateRandomBytes(256)
		if err != nil {
			return key, errors.Wrap(err, "failed to generate JWT encryption key")
		}
		err := storage.SetValue(encryptionKeyBucket, encryptionKeyName, key)
		if err != nil {
			return key, errors.Wrap(err, "failed to store JWT encryption key")
		}
	}

	return key, nil
}

func generateRandomBytes(length int) ([]byte, error) {
	key := make([]byte, length)
	_, err := rand.Read(key)

	if err != nil {
		return nil, err
	}

	return key, nil
}
