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

package logconfig

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestCollector_List_ListsAllLogFilesMatchingPattern(t *testing.T) {
	assert := assert.New(t)

	// given
	baseName := "mysterium-test.log"
	fn1 := NewTempFileName(t, baseName)
	defer func() { _ = os.Remove(fn1) }()
	fn2 := NewTempFileName(t, baseName)
	defer func() { _ = os.Remove(fn2) }()

	opts := LogOptions{
		LogLevel: zerolog.DebugLevel,
		Filepath: path.Join(path.Dir(fn1), baseName),
	}
	collector := NewCollector(&opts)

	// when
	logFiles, err := collector.logFilepaths()

	// then
	assert.NoError(err)
	assert.Contains(logFiles, fn1)
	assert.Contains(logFiles, fn2)
}

func TestCollector_Archive(t *testing.T) {
	assert := assert.New(t)

	// given
	baseName := "mysterium-test.log"
	fn1 := NewTempFileName(t, baseName)
	defer func() { _ = os.Remove(fn1) }()
	fn2 := NewTempFileName(t, baseName)
	defer func() { _ = os.Remove(fn2) }()

	opts := LogOptions{
		LogLevel: zerolog.DebugLevel,
		Filepath: path.Join(path.Dir(fn1), baseName),
	}
	collector := NewCollector(&opts)

	// when
	zipFilename, err := collector.Archive()
	defer func() { _ = os.Remove(zipFilename) }()

	// then
	assert.NoError(err)
	assert.NotEmpty(zipFilename)
}

func NewTempFileName(t *testing.T, pattern string) string {
	file, err := ioutil.TempFile("", pattern)
	assert.NoError(t, err)
	return file.Name()
}
