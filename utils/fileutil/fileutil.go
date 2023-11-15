/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

// Copyright 2018 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package fileutil provides utility methods used when dealing with the filesystem in tsdb.
// It is largely copied from github.com/coreos/etcd/pkg/fileutil to avoid the
// dependency chain it brings with it.
// Please check github.com/coreos/etcd for licensing information.

package fileutil

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CopyDirs copies all directories, subdirectories and files recursively including the empty folders.
// Source and destination must be full paths.
func CopyDirs(src, dest string) error {
	if err := os.MkdirAll(dest, 0777); err != nil {
		return err
	}
	files, err := readDirs(src)
	if err != nil {
		return err
	}

	for _, f := range files {
		dp := filepath.Join(dest, f)
		sp := filepath.Join(src, f)

		stat, err := os.Stat(sp)
		if err != nil {
			return err
		}

		// Empty directories are also created.
		if stat.IsDir() {
			if err := os.MkdirAll(dp, 0777); err != nil {
				return err
			}
			continue
		}

		if err := copyFile(sp, dp); err != nil {
			return err
		}
	}
	return nil
}

// copyFile copies a file from `src` to `dest`.
// It tries to also copy it with the same permissions,
// if thats not possible 0644 is used.
func copyFile(src, dest string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	perm := os.FileMode(0644)
	if info, err := os.Stat(src); err == nil {
		perm = info.Mode().Perm()
	}

	err = os.WriteFile(dest, data, perm)
	if err != nil {
		return err
	}
	return nil
}

// readDirs reads the source directory recursively and
// returns relative paths to all files and empty directories.
func readDirs(src string) ([]string, error) {
	var files []string

	err := filepath.Walk(src, func(path string, f os.FileInfo, err error) error {
		relativePath := strings.TrimPrefix(path, src)
		if len(relativePath) > 0 {
			files = append(files, relativePath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// ReadDir returns the filenames in the given directory in sorted order.
func ReadDir(dirpath string) ([]string, error) {
	dir, err := os.Open(dirpath)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	names, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}
