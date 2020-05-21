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

package dev

import (
	"runtime"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/mysteriumnetwork/node/ci/packages"
)

// Daemon builds and runs myst daemon
func Daemon() error {
	mg.Deps(packages.Build)

	cmd := "build/myst/myst daemon"
	if runtime.GOOS == "darwin" {
		cmd = "sudo " + cmd
	}
	cmdParts := strings.Split(cmd, " ")
	return sh.RunV(cmdParts[0], cmdParts[1:]...)
}

// Openvpn builds and starts openvpn service with terms accepted
func Openvpn() error {
	mg.Deps(packages.Build)

	cmd := "build/myst/myst service --agreed-terms-and-conditions openvpn"
	if runtime.GOOS == "darwin" {
		cmd = "sudo " + cmd
	}
	cmdParts := strings.Split(cmd, " ")
	return sh.RunV(cmdParts[0], cmdParts[1:]...)
}

// Wireguard builds and starts wireguard service with terms accepted
func Wireguard() error {
	mg.Deps(packages.Build)

	cmd := "build/myst/myst service --agreed-terms-and-conditions wireguard"
	if runtime.GOOS == "darwin" {
		cmd = "sudo " + cmd
	}
	cmdParts := strings.Split(cmd, " ")
	return sh.RunV(cmdParts[0], cmdParts[1:]...)
}

// CLI builds and runs myst CLI
func CLI() error {
	mg.Deps(packages.Build)

	return sh.RunV("build/myst/myst", "cli")
}
