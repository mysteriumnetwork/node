/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package storage

import "github.com/asdine/storm"

// FakeStorage for testing
type FakeStorage struct{}

// Store for testing
func (fs *FakeStorage) Store(issuer string, data interface{}) error { return nil }

// Delete for testing
func (fs *FakeStorage) Delete(issuer string, data interface{}) error { return nil }

// Close for testing
func (fs *FakeStorage) Close() error { return nil }

// GetAll for testing
func (fs *FakeStorage) GetAll(issuer string, data interface{}) error { return nil }

// GetDB for testing
func (fs *FakeStorage) GetDB() *storm.DB { return nil }
