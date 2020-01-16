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
	"github.com/mysteriumnetwork/node/e2e"
	"github.com/mysteriumnetwork/node/logconfig"
)

// TestE2EBasic runs end-to-end tests
func TestE2EBasic() error {
	logconfig.Bootstrap()
	composeFiles := []string{
		"./docker-compose.e2e-basic.yml",
	}
	runner, cleanup := e2e.NewRunner(composeFiles, "node_e2e_basic_test", "openvpn,noop,wireguard")
	defer cleanup()
	if err := runner.Init(); err != nil {
		return err
	}
	return runner.Test()
}

// TestE2ENAT runs end-to-end tests in NAT environment
func TestE2ENAT() error {
	logconfig.Bootstrap()
	composeFiles := []string{
		"./docker-compose.e2e-traversal.yml",
	}
	runner, cleanup := e2e.NewRunner(composeFiles, "node_e2e_nat_test", "openvpn")
	defer cleanup()
	if err := runner.Init(); err != nil {
		return err
	}
	return runner.Test()
}
