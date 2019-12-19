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
	"path"
	"strings"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
)

// Collector collects node logs.
type Collector struct {
	options *LogOptions
}

// NewCollector creates a Collector instance.
func NewCollector(options *LogOptions) *Collector {
	return &Collector{options: options}
}

// Archive creates ZIP archive containing all node log files.
func (c *Collector) Archive() (outputFilepath string, err error) {
	if c.options.Filepath == "" {
		return "", errors.New("file logging is disabled, can't retrieve logs")
	}
	filepaths, err := c.logFilepaths()
	if err != nil {
		return "", err
	}

	zip := archiver.NewZip()
	zip.OverwriteExisting = true

	zipFilepath := c.options.Filepath + ".zip"
	err = zip.Archive(filepaths, zipFilepath)
	if err != nil {
		return "", errors.Wrap(err, "could not create log archive")
	}

	return zipFilepath, nil
}

func (c *Collector) logFilepaths() (result []string, err error) {
	filename := path.Base(c.options.Filepath)
	dir := path.Dir(c.options.Filepath)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read directory: "+dir)
	}
	for _, f := range files {
		if strings.Contains(f.Name(), filename) {
			result = append(result, path.Join(dir, f.Name()))
		}
	}
	return result, nil
}
