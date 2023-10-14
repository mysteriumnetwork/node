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
	"os"
	"path"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestCollector_List_ListsAllLogFilesMatchingPattern(t *testing.T) {
	assert := assert.New(t)

	// given
	baseName := "mysterium-test"
	dn1 := NewTempDirName(t, "")
	logFilename := baseName + ".log"
	f1, err := os.Create(path.Join(dn1, logFilename))
	assert.NoError(err)
	defer os.Remove(f1.Name())

	fn2 := NewTempFileName(t, dn1, logFilename+".gz")
	defer os.Remove(fn2)

	fn3 := NewTempFileName(t, dn1, logFilename+".gz")
	defer os.Remove(fn3)

	// ensure this is the file with the most recent modified time
	time.Sleep(time.Millisecond * 10)
	fn4 := NewTempFileName(t, dn1, logFilename+".gz")
	defer os.Remove(fn4)

	opts := LogOptions{
		LogLevel: zerolog.DebugLevel,
		Filepath: path.Join(path.Dir(f1.Name()), baseName),
	}
	collector := NewCollector(&opts)

	// when
	logFiles, err := collector.logFilepaths()

	// then
	assert.NoError(err)
	assert.Contains(logFiles, f1.Name())
	assert.Contains(logFiles, fn4)
	assert.Len(logFiles, 2)
}

func TestCollector_Archive(t *testing.T) {
	assert := assert.New(t)

	// given
	baseName := "mysterium-test"
	dn1 := NewTempDirName(t, "")
	logFilename := baseName + ".log"
	f1, err := os.Create(path.Join(dn1, logFilename))
	assert.NoError(err)
	defer os.Remove(f1.Name())

	fn2 := NewTempFileName(t, dn1, logFilename+".gz")
	defer os.Remove(fn2)

	opts := LogOptions{
		LogLevel: zerolog.DebugLevel,
		Filepath: path.Join(path.Dir(f1.Name()), baseName),
	}
	collector := NewCollector(&opts)

	// when
	zipFilename, err := collector.Archive()
	defer os.Remove(zipFilename)

	// then
	assert.NoError(err)
	assert.NotEmpty(zipFilename)
}

func NewTempFileName(t *testing.T, dir, pattern string) string {
	file, err := os.CreateTemp(dir, pattern)
	assert.NoError(t, err)
	return file.Name()
}

func NewTempDirName(t *testing.T, pattern string) string {
	dir, err := os.MkdirTemp("", pattern)
	assert.NoError(t, err)
	return dir
}
