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
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mysteriumnetwork/node/config"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

// CredentialsManager verifies/sets user credentials for web UI.
type CredentialsManager struct {
	fileLocation string
}

const passwordFile = "nodeui-pass"

var (
	// ErrBadCredentials represents an error when validating wrong c.
	ErrBadCredentials = errors.New("bad credentials")

	passwordNotFound = errors.New("password or password file doesn't exist")
)

// NewCredentialsManager returns given a password file directory
// returns a new credentials manager, which can be used to validate or alter
// user credentials.
func NewCredentialsManager(dataDir string) *CredentialsManager {
	return &CredentialsManager{
		fileLocation: filepath.Join(dataDir, passwordFile),
	}
}

// Validate username and password against stored credentials.
func (c *CredentialsManager) Validate(username, password string) error {
	if username != config.FlagTequilapiUsername.Value {
		return ErrBadCredentials
	}

	storedHash, err := c.loadOrInitialize()
	if err != nil {
		return fmt.Errorf("could not load credentials: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	if err != nil {
		return fmt.Errorf("bad credentials: %w", err)
	}

	return nil
}

// SetPassword sets a new password for a user.
func (c *CredentialsManager) SetPassword(password string) error {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("unable to generate password hash: %w", err)
	}

	return c.setPassword(string(passwordHash))
}

func (c *CredentialsManager) loadOrInitialize() (string, error) {
	storedHash, err := c.getPassword()
	if !errors.Is(err, passwordNotFound) {
		return storedHash, err
	}

	log.Info().Err(err).Msg("CredentialsManager not found, initializing to default")

	err = c.SetPassword(config.FlagTequilapiPassword.Value)
	if err != nil {
		return "", fmt.Errorf("failed to set initial credentials: %w", err)
	}

	return c.getPassword()
}

func (c *CredentialsManager) setPassword(passwordHash string) error {
	f, err := os.OpenFile(c.fileLocation, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("could not open password file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(passwordHash); err != nil {
		return fmt.Errorf("could not write the new password: %w", err)
	}

	return f.Sync()
}

func (c *CredentialsManager) getPassword() (string, error) {
	file, err := os.Open(c.fileLocation)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", passwordNotFound
		}

		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		return scanner.Text(), scanner.Err()
	}

	return "", passwordNotFound
}
