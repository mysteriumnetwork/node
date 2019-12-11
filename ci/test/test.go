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

package test

import (
	"strings"

	"github.com/magefile/mage/sh"
)

// Test runs unit tests
func Test() error {
	packages, err := unitTestPackages()
	if err != nil {
		return err
	}
	args := append([]string{"test", "-race", "-timeout", "3m", "-cover", "-coverprofile", "coverage.txt", "-covermode", "atomic"}, packages...)
	return sh.RunV("go", args...)
}

func unitTestPackages() ([]string, error) {
	allPackages, err := listPackages()
	if err != nil {
		return nil, err
	}
	packages := make([]string, 0)
	for _, p := range allPackages {
		if !strings.Contains(p, "e2e") {
			packages = append(packages, p)
		}
	}
	return packages, nil
}

func listPackages() ([]string, error) {
	output, err := sh.Output("go", "list", "./...")
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.Replace(output, "\r\n", "\n", -1), "\n"), nil
}
