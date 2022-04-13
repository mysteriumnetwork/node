/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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

package contract

// MigrationStatus status of the migration
type MigrationStatus = string

const (
	// MigrationStatusRequired means new there is new Hermes and identity required to migrate to it
	MigrationStatusRequired = "required"
	// MigrationStatusFinished means migration to new Hermes finished or not needed
	MigrationStatusFinished = "finished"
)

// MigrationStatusResponse represents status of the migration
// swagger:model MigrationStatusResponse
type MigrationStatusResponse struct {
	Status MigrationStatus `json:"status"`
}
