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
	"github.com/asdine/storm"
)

type storage struct {
	db *storm.DB
}

// OpenDB creates new or open existing BoltDB
func OpenDB(name string) (*storage, error) {
	db, err := storm.Open(name)
	return &storage{db}, err
}

// Store allows to keep promises grouped by the issuer
func (s *storage) Store(issuer string, data interface{}) error {
	return s.db.From(issuer).Save(data)
}

// GetAll allows to get all promises by the issuer
func (s *storage) GetAll(issuer string, data interface{}) error {
	return s.db.From(issuer).All(data)
}

// Delete removes promise record from the database
func (s *storage) Delete(issuer string, data interface{}) error {
	return s.db.From(issuer).DeleteStruct(data)
}

// Close closes database
func (s *storage) Close() error {
	return s.db.Close()
}
