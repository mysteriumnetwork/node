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

// Package boltdbtest contains the utitilies needed for boltdb testing
package boltdbtest

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/asdine/storm"
	"github.com/stretchr/testify/assert"
)

// CreateDB creates a new boltdb instance with a temp file and returns the underlying file and database
func CreateDB(t *testing.T) (string, *storm.DB) {
	file := CreateTempFile(t)
	db, err := storm.Open(file)
	assert.Nil(t, err)

	return file, db
}

// CleanupDB removes the provided file after stopping the given db
func CleanupDB(t *testing.T, fileName string, db *storm.DB) {
	err := db.Close()
	if err != nil {
		t.Logf("Could not close db %v\n", err)
	}
	CleanupTempFile(t, fileName)
}

// CreateTempFile creates a temporary file and returns its path
func CreateTempFile(t *testing.T) string {
	file, err := ioutil.TempFile("", "")
	assert.Nil(t, err)
	return file.Name()
}

// CleanupTempFile removes the given temp file
func CleanupTempFile(t *testing.T, fileName string) {
	err := os.Remove(fileName)
	if err != nil {
		t.Logf("Could not remove temp file: %v. Err: %v\n", fileName, err)
	}
}

// CreateTempDir creates a temporary directory
func CreateTempDir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "")
	assert.Nil(t, err)
	return dir
}

// RemoveTempDir removes a temporary directory
func RemoveTempDir(t *testing.T, dir string) {
	err := os.RemoveAll(dir)
	if err != nil {
		t.Logf("Could not remove temp dir: %v. Err: %v\n", dir, err)
	}
}
