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
	"github.com/mysteriumnetwork/node/core/storage"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

// Storage for Credentials
type Storage interface {
	StoreOrUpdate(bucket string, data interface{}) error
	GetOneByField(bucket string, fieldName string, key interface{}, to interface{}) error
}

const (
	initialUsername          = "myst"
	initialPassword          = "mystberry"
	credentialsStorageBucket = "app-credentials"
	credentialsStorageID     = 1
)

// Credentials verifies/sets user credentials for web UI
type Credentials struct {
	username, password string
	db                 Storage
}

// storeCredentials - struct stored in DB
type storeCredentials struct {
	ID           int
	Username     string
	PasswordHash string
}

// NewCredentials instance
func NewCredentials(username, password string, db Storage) *Credentials {
	return &Credentials{
		username: username,
		password: password,
		db:       db,
	}
}

// Validate username and password against stored Credentials
func (credentials *Credentials) Validate() (err error) {
	var sc *storeCredentials
	sc, err = credentials.loadOrInitialize()
	if err != nil {
		return errors.Wrap(err, "could not load credentials")
	}
	if credentials.username != sc.Username {
		return errors.New("bad credentials")
	}
	err = bcrypt.CompareHashAndPassword([]byte(sc.PasswordHash), []byte(credentials.password))
	if err != nil {
		return errors.Wrap(err, "bad credentials")
	}
	return nil
}

func (credentials *Credentials) loadOrInitialize() (s *storeCredentials, err error) {
	var sc = &storeCredentials{}
	err = credentials.db.GetOneByField(credentialsStorageBucket, "ID", credentialsStorageID, sc)
	if err == storage.ErrNotFound {
		err = NewCredentials(initialUsername, initialPassword, credentials.db).Set()
		if err != nil {
			return nil, errors.Wrap(err, "failed to set initial credentials")
		}
		err = credentials.db.GetOneByField(credentialsStorageBucket, "ID", credentialsStorageID, sc)
	}
	return sc, err
}

// Set new Credentials
func (credentials *Credentials) Set() (err error) {
	var passwordHash []byte
	passwordHash, err = bcrypt.GenerateFromPassword([]byte(initialPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.Wrap(err, "unable to generate password hash")
	}
	sc := &storeCredentials{
		ID:           credentialsStorageID,
		Username:     credentials.username,
		PasswordHash: string(passwordHash),
	}
	err = credentials.db.StoreOrUpdate(credentialsStorageBucket, sc)
	if err != nil {
		return errors.Wrap(err, "unable to set credentials")
	}
	return nil
}
