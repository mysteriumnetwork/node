/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package boltdb

import (
	"path/filepath"

	"github.com/asdine/storm"
)

// Bolt is a wrapper around boltdb
type Bolt struct {
	db *storm.DB
}

// NewStorage creates a new BoltDB storage for service promises
func NewStorage(path string) (*Bolt, error) {
	return openDB(filepath.Join(path, "myst.db"))
}

// openDB creates new or open existing BoltDB
func openDB(name string) (*Bolt, error) {
	db, err := storm.Open(name)
	if err != nil {
		return nil, err
	}

	bolt := &Bolt{db}
	migrator := NewMigrator(bolt)
	err = migrator.Up()
	return bolt, err
}

// Store allows to keep struct grouped by the bucket
func (b *Bolt) Store(bucket string, data interface{}) error {
	return b.db.From(bucket).Save(data)
}

// GetAllFrom allows to get all structs from the bucket
func (b *Bolt) GetAllFrom(bucket string, data interface{}) error {
	return b.db.From(bucket).All(data)
}

// Delete removes the given struct from the given bucket
func (b *Bolt) Delete(bucket string, data interface{}) error {
	return b.db.From(bucket).DeleteStruct(data)
}

// Update allows to update the struct in the given bucket
func (b *Bolt) Update(bucket string, object interface{}) error {
	return b.db.From(bucket).Update(object)
}

// Close closes database
func (b *Bolt) Close() error {
	return b.db.Close()
}
