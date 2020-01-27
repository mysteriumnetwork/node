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
	"fmt"

	"github.com/mysteriumnetwork/node/e2e"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/rs/zerolog/log"
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
	return runner.Test("myst-provider", "myst-consumer")
}

// TestE2ENAT runs end-to-end tests in NAT environment
func TestE2ENAT() error {
	logconfig.Bootstrap()
	composeFiles := []string{
		"./docker-compose.e2e-traversal.yml",
	}
	runner, cleanup := e2e.NewRunner(composeFiles, "node_e2e_nat_test", "wireguard,openvpn")
	defer cleanup()
	if err := runner.Init(); err != nil {
		return err
	}
	return runner.Test("myst-provider", "myst-consumer")
}

// TestE2ECompatibility runs end-to-end tests with older node version to make check compatibility
func TestE2ECompatibility() error {
	logconfig.Bootstrap()
	composeFiles := []string{
		"./docker-compose.e2e-compatibility.yml",
	}
	runner, cleanup := e2e.NewRunner(composeFiles, "node_e2e_compatibility", "openvpn,noop,wireguard")
	defer cleanup()
	if err := runner.Init(); err != nil {
		return err
	}

	tests := []struct {
		provider, consumer string
	}{
		{provider: "myst-provider", consumer: "myst-consumer-local"},
		{provider: "myst-provider-local", consumer: "myst-consumer"},
	}

	for _, test := range tests {
		log.Info().Msgf("Testing compatibility for %s and %s", test.provider, test.consumer)

		if err := runner.Test(test.provider, test.consumer); err != nil {
			return fmt.Errorf("compatibility test failed for %s and %s", test.provider, test.consumer)
		}
	}
	return nil
}
