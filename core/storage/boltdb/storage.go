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
	"sync"

	"github.com/asdine/storm/v3"
	"github.com/pkg/errors"
)

// Bolt is a wrapper around boltdb
type Bolt struct {
	mux sync.RWMutex
	db  *storm.DB
}

// NewStorage creates a new BoltDB storage for service promises
func NewStorage(path string) (*Bolt, error) {
	return openDB(filepath.Join(path, "myst.db"))
}

// openDB creates new or open existing BoltDB
func openDB(name string) (*Bolt, error) {
	db, err := storm.Open(name)
	return &Bolt{
		db: db,
	}, errors.Wrap(err, "failed to open boltDB")
}

// GetValue gets key value
func (b *Bolt) GetValue(bucket string, key interface{}, to interface{}) error {
	b.mux.RLock()
	defer b.mux.RUnlock()
	return b.db.Get(bucket, key, to)
}

// SetValue sets key value
func (b *Bolt) SetValue(bucket string, key interface{}, to interface{}) error {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.db.Set(bucket, key, to)
}

// Store allows to keep struct grouped by the bucket
func (b *Bolt) Store(bucket string, data interface{}) error {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.db.From(bucket).Save(data)
}

// GetAllFrom allows to get all structs from the bucket
func (b *Bolt) GetAllFrom(bucket string, data interface{}) error {
	b.mux.RLock()
	defer b.mux.RUnlock()
	return b.db.From(bucket).All(data)
}

// Delete removes the given struct from the given bucket
func (b *Bolt) Delete(bucket string, data interface{}) error {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.db.From(bucket).DeleteStruct(data)
}

// DeleteKey the given struct from the given bucket
func (b *Bolt) DeleteKey(bucket string, key interface{}) error {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.db.Delete(bucket, key)
}

// Update allows to update the struct in the given bucket
func (b *Bolt) Update(bucket string, object interface{}) error {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.db.From(bucket).Update(object)
}

// GetOneByField returns an object from the given bucket by the given field
func (b *Bolt) GetOneByField(bucket string, fieldName string, key interface{}, to interface{}) error {
	b.mux.RLock()
	defer b.mux.RUnlock()
	return b.db.From(bucket).One(fieldName, key, to)
}

// GetLast returns the last entry in the bucket
func (b *Bolt) GetLast(bucket string, to interface{}) error {
	b.mux.RLock()
	defer b.mux.RUnlock()
	return b.db.From(bucket).Select().Reverse().First(to)
}

// GetBuckets returns a list of buckets
func (b *Bolt) GetBuckets() []string {
	b.mux.RLock()
	defer b.mux.RUnlock()
	return b.db.Bucket()
}

// DB returns raw storm DB.
func (b *Bolt) DB() *storm.DB {
	return b.db
}

// Close closes database
func (b *Bolt) Close() error {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.db.Close()
}

// RLock locks underlying RWMutex for reading
func (b *Bolt) RLock() {
	b.mux.RLock()
}

// RUnlock unlocks underlying RWMutex for reading
func (b *Bolt) RUnlock() {
	b.mux.RUnlock()
}

// Lock locks underlying RWMutex for write
func (b *Bolt) Lock() {
	b.mux.Lock()
}

// Unlock unlocks underlying RWMutex for write
func (b *Bolt) Unlock() {
	b.mux.Unlock()
}
