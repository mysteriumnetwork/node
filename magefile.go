// +build mage

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

package main

import (
	"github.com/mysteriumnetwork/go-ci/env"
	// mage:import
	_ "github.com/mysteriumnetwork/node/ci/check"
	// mage:import
	_ "github.com/mysteriumnetwork/node/ci/dev"
	// mage:import
	_ "github.com/mysteriumnetwork/node/ci/test"
	// mage:import
	_ "github.com/mysteriumnetwork/node/ci/packages"
	// mage:import
	_ "github.com/mysteriumnetwork/node/ci/release"
	// mage:import
	_ "github.com/mysteriumnetwork/node/ci/storage"
	// mage:import
	_ "github.com/mysteriumnetwork/node/ci/util/docker"
	// mage:import
	_ "github.com/mysteriumnetwork/node/localnet"
)

// GenerateEnvFile generates env file for further stages
func GenerateEnvFile() error {
	return env.GenerateEnvFile()
}
