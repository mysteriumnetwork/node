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
	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/core/storage"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

// Storage for Credentials
type Storage interface {
	GetValue(bucket string, key interface{}, to interface{}) error
	SetValue(bucket string, key interface{}, to interface{}) error
}

const (
	username            = "myst"
	initialPassword     = "mystberry"
	credentialsDBBucket = "app-credentials"
)

// Credentials verifies/sets user credentials for web UI
type Credentials struct {
	username, password string
	db                 Storage
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
	var storedHash string
	storedHash, err = credentials.loadOrInitialize()
	if err != nil {
		return errors.Wrap(err, "could not load credentials")
	}
	if credentials.username != username {
		return errors.New("bad credentials")
	}
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(credentials.password))
	if err != nil {
		return errors.Wrap(err, "bad credentials")
	}
	return nil
}

func (credentials *Credentials) loadOrInitialize() (s string, err error) {
	var storedHash string
	err = credentials.db.GetValue(credentialsDBBucket, username, &storedHash)
	if err == storage.ErrNotFound {
		log.Info("[web-ui-auth] credentials not found, initializing to default")
		err = NewCredentials(username, initialPassword, credentials.db).Set()
		if err != nil {
			return "", errors.Wrap(err, "failed to set initial credentials")
		}
		err = credentials.db.GetValue(credentialsDBBucket, username, &storedHash)
	}
	return storedHash, err
}

// Set new Credentials
func (credentials *Credentials) Set() (err error) {
	var passwordHash []byte
	passwordHash, err = bcrypt.GenerateFromPassword([]byte(credentials.password), bcrypt.DefaultCost)
	if err != nil {
		return errors.Wrap(err, "unable to generate password hash")
	}
	err = credentials.db.SetValue(credentialsDBBucket, credentials.username, string(passwordHash))
	if err != nil {
		return errors.Wrap(err, "unable to set credentials")
	}
	return nil
}
