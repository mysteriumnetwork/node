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
	"errors"
	"fmt"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/storage"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

// Storage for Credentials.
type Storage interface {
	GetValue(bucket string, key interface{}, to interface{}) error
	SetValue(bucket string, key interface{}, to interface{}) error
}

const credentialsDBBucket = "app-credentials"

// ErrBadCredentials represents an error when validating wrong credentials.
var ErrBadCredentials = errors.New("bad credentials")

// Credentials verifies/sets user credentials for web UI.
type Credentials struct {
	username, password string
	db                 Storage
}

// NewCredentials instance.
func NewCredentials(username, password string, db Storage) *Credentials {
	return &Credentials{
		username: username,
		password: password,
		db:       db,
	}
}

// Validate username and password against stored Credentials.
func (credentials *Credentials) Validate() (err error) {
	if credentials.username != config.FlagTequilapiUsername.Value {
		return ErrBadCredentials
	}

	storedHash, err := credentials.loadOrInitialize()
	if err != nil {
		return fmt.Errorf("could not load credentials: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(credentials.password))
	if err != nil {
		return fmt.Errorf("bad credentials: %w", err)
	}

	return nil
}

func (credentials *Credentials) loadOrInitialize() (s string, err error) {
	var storedHash string

	err = credentials.db.GetValue(credentialsDBBucket, config.FlagTequilapiUsername.Value, &storedHash)
	if !errors.Is(err, storage.ErrNotFound) {
		return storedHash, err
	}

	log.Info().Msg("Credentials not found, initializing to default")

	err = NewCredentials(config.FlagTequilapiUsername.Value, config.FlagTequilapiPassword.Value, credentials.db).Set()
	if err != nil {
		return "", fmt.Errorf("failed to set initial credentials: %w", err)
	}

	err = credentials.db.GetValue(credentialsDBBucket, config.FlagTequilapiUsername.Value, &storedHash)

	return storedHash, err
}

// Set new credentials.
func (credentials *Credentials) Set() (err error) {
	var passwordHash []byte

	passwordHash, err = bcrypt.GenerateFromPassword([]byte(credentials.password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("unable to generate password hash: %w", err)
	}

	err = credentials.db.SetValue(credentialsDBBucket, credentials.username, string(passwordHash))
	if err != nil {
		return fmt.Errorf("unable to set credentials: %w", err)
	}

	return nil
}
