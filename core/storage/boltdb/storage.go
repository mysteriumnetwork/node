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
	"github.com/mysteriumnetwork/node/core/storage"
)

type bolt struct {
	db *storm.DB
}

// NewStorage creates a new BoltDB storage for service promises
func NewStorage(path string) (storage.Storage, error) {
	return openDB(filepath.Join(path, "myst.db"))
}

// openDB creates new or open existing BoltDB
func openDB(name string) (*bolt, error) {
	db, err := storm.Open(name)
	return &bolt{db}, err
}

// Store allows to keep promises grouped by the issuer
func (b *bolt) Store(issuer string, data interface{}) error {
	return b.db.From(issuer).Save(data)
}

// GetAll allows to get all promises by the issuer
func (b *bolt) GetAll(issuer string, data interface{}) error {
	return b.db.From(issuer).All(data)
}

// Delete removes promise record from the database
func (b *bolt) Delete(issuer string, data interface{}) error {
	return b.db.From(issuer).DeleteStruct(data)
}

// Close closes database
func (b *bolt) Close() error {
	return b.db.Close()
}

func (b *bolt) StoreSession(bucketName string, key string, value interface{}) error {
	if err := b.db.Set(bucketName, key, value); err != nil {
		return err
	}
	return nil
}

// GetAll allows to get all promises by the issuer
//func (b *bolt) GetSessions(bucketName string, data interface{}) error {
//	return b.db.From(issuer).All(data)
//}
