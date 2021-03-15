/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package migrations

import (
	"time"

	"github.com/asdine/storm/v3"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
)

type storedRegistrationStatus struct {
	ID                  string `storm:"id"`
	RegistrationStatus  registry.RegistrationStatus
	Identity            identity.Identity
	ChainID             int64
	RegistrationRequest registry.IdentityRegistrationRequest
	UpdatedAt           time.Time
}

const registrationStatusBucket = "registry_statuses"

// MigrateRegistrationState run a migration which resets all registration
// states to "Unregistered".
func MigrateRegistrationState(db *storm.DB) error {
	tx, err := db.Begin(true)
	if err != nil {
		return err
	}
	res := []storedRegistrationStatus{}
	err = tx.From(registrationStatusBucket).All(&res)
	if err != nil {
		return err
	}

	for i := range res {
		res[i].RegistrationStatus = registry.Unregistered
		err := tx.From(registrationStatusBucket).Save(&res[i])
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
