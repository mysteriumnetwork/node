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

package boltdb

import (
	"path/filepath"

	"github.com/asdine/storm"
	"github.com/pkg/errors"
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
	return &Bolt{db}, errors.Wrap(err, "failed to open boltDB")
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

// GetOneByField returns an object from the given bucket by the given field
func (b *Bolt) GetOneByField(bucket string, fieldName string, key interface{}, to interface{}) error {
	return b.db.From(bucket).One(fieldName, key, to)
}

// GetLast returns the last entry in the bucket
func (b *Bolt) GetLast(bucket string, to interface{}) error {
	return b.db.From(bucket).Select().Reverse().First(to)
}

// GetBuckets returns a list of buckets
func (b *Bolt) GetBuckets() []string {
	return b.db.Bucket()
}

// Close closes database
func (b *Bolt) Close() error {
	return b.db.Close()
}
